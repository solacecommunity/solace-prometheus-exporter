package main

import (
	"net/http"
	"os"
	"solace_exporter/internal/exporter"
	"solace_exporter/internal/web"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	promVersion "github.com/prometheus/common/version"
	"golang.org/x/sync/semaphore"
)

func logDataSource(dataSources []exporter.DataSource) string {
	dS := make([]string, len(dataSources))
	for index, dataSource := range dataSources {
		dS[index] = dataSource.String()
	}
	return strings.Join(dS, "&")
}

func main() {
	kingpin.HelpFlag.Short('h')

	promlogConfig := promlog.Config{
		Level:  &promlog.AllowedLevel{},
		Format: &promlog.AllowedFormat{},
	}
	promlogConfig.Level.Set("info")
	promlogConfig.Format.Set("logfmt")
	flag.AddFlags(kingpin.CommandLine, &promlogConfig)

	configFile := kingpin.Flag(
		"config-file",
		"Path and name of ini file with configuration settings. See sample file solace_prometheus_exporter.ini.",
	).String()
	enableTLS := kingpin.Flag(
		"enable-tls",
		"Set to true, to start listenAddr as TLS port. Make sure to provide valid server certificate and private key files.",
	).Bool()
	certfile := kingpin.Flag(
		"certificate",
		"If using TLS, you must provide a valid server certificate in PEM format. Can be set via config file, cli parameter or env variable.",
	).ExistingFile()
	privateKey := kingpin.Flag(
		"private-key",
		"If using TLS, you must provide a valid private key in PEM format. Can be set via config file, cli parameter or env variable.",
	).ExistingFile()
	certType := kingpin.Flag(
		"cert-type",
		" Set the certificate type PEM | PKCS12.",
	).String()
	pkcs12File := kingpin.Flag(
		"pkcs12File",
		"If using TLS, you must provide a valid pkcs12 file.",
	).ExistingFile()
	pkcs12Pass := kingpin.Flag(
		"pkcs12Pass",
		"If using TLS, you must provide a valid pkcs12 password.",
	).String()
	kingpin.Parse()

	logger := promlog.New(&promlogConfig)

	endpoints, conf, err := exporter.ParseConfig(*configFile)
	if err != nil {
		level.Error(logger).Log("msg", err)
		os.Exit(1)
	}

	level.Info(logger).Log("msg", "Starting solace_prometheus_exporter")
	level.Info(logger).Log("msg", "Build context", "context", promVersion.BuildContext())

	if *enableTLS {
		conf.EnableTLS = *enableTLS
	}
	if len(*certfile) > 0 {
		conf.Certificate = *certfile
	}
	if len(*privateKey) > 0 {
		conf.PrivateKey = *privateKey
	}
	if len(*certType) > 0 {
		conf.CertType = *certType
	}
	if len(*pkcs12File) > 0 {
		conf.Pkcs12File = *pkcs12File
	}
	if len(*pkcs12Pass) > 0 {
		conf.Pkcs12Pass = *pkcs12Pass
	}
	level.Info(logger).Log("msg", "Scraping",
		"listenAddr", conf.GetListenURI(),
		"scrapeURI", conf.ScrapeURI,
		"username", conf.Username,
		"sslVerify", conf.SslVerify,
		"timeout", conf.Timeout)

	// Configure endpoints
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		doHandle(w, r, nil, *conf, logger)
	})

	// A broker has only max 10 semp connections that can be served in parallel.
	var sempConnections = semaphore.NewWeighted(conf.ParallelSempConnections)
	declareHandlerFromConfig := func(urlPath string, dataSource []exporter.DataSource) {
		_ = level.Info(logger).Log("msg", "Register handler from config", "handler", "/"+urlPath, "dataSource", logDataSource(dataSource))

		if conf.PrefetchInterval.Seconds() > 0 {
			var asyncFetcher = exporter.NewAsyncFetcher(urlPath, dataSource, *conf, logger, sempConnections)
			http.HandleFunc("/"+urlPath, func(w http.ResponseWriter, r *http.Request) {
				doHandleAsync(w, r, asyncFetcher)
			})
		} else {
			http.HandleFunc("/"+urlPath, func(w http.ResponseWriter, r *http.Request) {
				doHandle(w, r, dataSource, *conf, logger)
			})
		}
	}
	for urlPath, dataSource := range endpoints {
		declareHandlerFromConfig(urlPath, dataSource)
	}

	http.HandleFunc("/solace", func(w http.ResponseWriter, r *http.Request) {
		var err = r.ParseForm()
		if err != nil {
			level.Error(logger).Log("msg", "Can not parse the request parameter", "err", err)
			return
		}

		var dataSource []exporter.DataSource
		for key, values := range r.Form {
			if strings.HasPrefix(key, "m.") {
				for _, value := range values {
					parts := strings.Split(value, "|")
					if len(parts) < 2 {
						level.Error(logger).Log("msg", "One or two | expected. Use VPN wildcard | Item wildcard | Optional metric filter for v2 apis", "key", key, "value", value)
					} else {
						var metricFilter []string
						if len(parts) == 3 && len(strings.TrimSpace(parts[2])) > 0 {
							metricFilter = strings.Split(parts[2], ",")
						}

						dataSource = append(dataSource, exporter.DataSource{
							Name:         strings.TrimPrefix(key, "m."),
							VpnFilter:    parts[0],
							ItemFilter:   parts[1],
							MetricFilter: metricFilter,
						})
					}
				}
			}
		}

		doHandle(w, r, dataSource, *conf, logger)
	})

	endpointViews := make([]web.EndpointView, 0, len(endpoints))

	for urlPath, dataSources := range endpoints {
		endpointViews = append(endpointViews, web.EndpointView{
			Path: urlPath,
			Meta: logDataSource(dataSources),
		})
	}

	handler, err := web.NewHandler(web.TemplateData{
		IsHWBroker: conf.IsHWBroker,
		Endpoints:  endpointViews,
	})
	if err != nil {
		level.Error(logger).Log("msg", err)
	}

	http.Handle("/", web.WrapWithAuth(handler, conf.ExporterAuth))

	// start server
	if conf.EnableTLS {
		exporter.ListenAndServeTLS(*conf)
	} else {
		server := &http.Server{
			Addr:              conf.ListenAddr,
			ReadHeaderTimeout: 5 * time.Second,
		}

		if err := server.ListenAndServe(); err != nil {
			level.Error(logger).Log("msg", "Error starting HTTP server", "err", err)
			os.Exit(2)
		}
	}
}

func doHandleAsync(w http.ResponseWriter, r *http.Request, asyncFetcher *exporter.AsyncFetcher) string {
	registry := prometheus.NewRegistry()
	registry.MustRegister(asyncFetcher)
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	handler.ServeHTTP(w, r)

	return w.Header().Get("status")
}

func doHandle(w http.ResponseWriter, r *http.Request, dataSource []exporter.DataSource, conf exporter.Config, logger log.Logger) string {
	var handler http.Handler
	if dataSource == nil {
		handler = promhttp.Handler()
	} else {
		// Exporter for endpoint
		username := r.FormValue("username")
		password := r.FormValue("password")
		scrapeURI := r.FormValue("scrapeURI")
		timeout := r.FormValue("timeout")
		if len(username) > 0 {
			conf.Username = username
		}
		if len(password) > 0 {
			conf.Password = password
		}
		if len(scrapeURI) > 0 {
			conf.ScrapeURI = scrapeURI
		}
		if len(timeout) > 0 {
			var err error
			conf.Timeout, err = time.ParseDuration(timeout)
			if err != nil {
				level.Error(logger).Log("msg", "Per HTTP given timeout parameter is not valid", "err", err, "timeout", timeout)
			}
		}

		level.Info(logger).Log("msg", "handle http request", "dataSource", logDataSource(dataSource), "scrapeURI", conf.ScrapeURI)

		exp := exporter.NewExporter(logger, &conf, &dataSource)
		registry := prometheus.NewRegistry()
		registry.MustRegister(exp)
		handler = promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	}
	securedHandler := web.WrapWithAuth(handler, conf.ExporterAuth)

	securedHandler.ServeHTTP(w, r)
	return w.Header().Get("status")
}
