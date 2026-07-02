package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"solace_exporter/internal/exporter"
	"solace_exporter/internal/web"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/promslog/flag"
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

	promlogConfig := promslog.Config{
		Level:  promslog.NewLevel(),
		Format: promslog.NewFormat(),
	}
	_ = promlogConfig.Level.Set("info")
	_ = promlogConfig.Format.Set("logfmt")
	flag.AddFlags(kingpin.CommandLine, &promlogConfig)

	configFile := kingpin.Flag(
		"config-file",
		"Path and name of ini file with configuration settings. See sample file solace_prometheus_exporter.ini.",
	).String()
	kingpin.Parse()

	logger := promslog.New(&promlogConfig)

	endpoints, conf, err := exporter.ParseConfig(*configFile)
	if err != nil {
		logger.Error("Error parsing config", "err", err)
		os.Exit(1)
	}

	logger.Info("Starting solace_prometheus_exporter")
	logger.Info("Build context", "context", promVersion.BuildContext())

	logger.Info("Scraping",
		"listenAddr", conf.GetListenURI(),
		"scrapeURI", conf.ScrapeURI,
		"username", conf.Username,
		"sslVerify", conf.SslVerify,
		"timeout", conf.Timeout)

	// Configure endpoints
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		doHandle(w, r, nil, conf, logger)
	})

	// A broker has only max 10 semp connections that can be served in parallel.
	var sempConnections = semaphore.NewWeighted(conf.ParallelSempConnections)
	declareHandlerFromConfig := func(urlPath string, dataSource []exporter.DataSource) {
		logger.Info("Register handler from config", "handler", "/"+urlPath, "dataSource", logDataSource(dataSource))

		if conf.PrefetchInterval.Seconds() > 0 {
			var asyncFetcher = exporter.NewAsyncFetcher(context.Background(), urlPath, dataSource, conf, logger, sempConnections)
			http.HandleFunc("/"+urlPath, func(w http.ResponseWriter, r *http.Request) {
				doHandleAsync(w, r, asyncFetcher, conf)
			})
		} else {
			http.HandleFunc("/"+urlPath, func(w http.ResponseWriter, r *http.Request) {
				doHandle(w, r, dataSource, conf, logger)
			})
		}
	}
	for urlPath, dataSource := range endpoints {
		declareHandlerFromConfig(urlPath, dataSource)
	}

	http.HandleFunc("/solace", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			logger.Error("Can not parse the request parameter", "err", err)
			return
		}

		doHandle(w, r, parseDataSources(r.Form, logger), conf, logger)
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
		logger.Error(err.Error())
	}

	http.Handle("/", web.WrapWithAuth(handler, conf.ExporterAuth))

	// start server
	if conf.EnableTLS {
		exporter.ListenAndServeTLS(conf)
	} else {
		server := &http.Server{
			Addr:              conf.ListenAddr,
			ReadHeaderTimeout: 5 * time.Second,
		}

		if err := server.ListenAndServe(); err != nil {
			logger.Error("Error starting HTTP server", "err", err)
			os.Exit(2)
		}
	}
}

func doHandleAsync(w http.ResponseWriter, r *http.Request, asyncFetcher *exporter.AsyncFetcher, conf *exporter.Config) string {
	registry := prometheus.NewRegistry()
	registry.MustRegister(asyncFetcher)
	// Protect prefetch endpoints with the same exporter auth as the synchronous handlers (they previously served
	// metrics unauthenticated even when SOLACE_EXPORTER_AUTH_* was configured).
	handler := web.WrapWithAuth(promhttp.HandlerFor(registry, promhttp.HandlerOpts{}), conf.ExporterAuth)
	handler.ServeHTTP(w, r)

	return w.Header().Get("status")
}

func doHandle(w http.ResponseWriter, r *http.Request, dataSource []exporter.DataSource, conf *exporter.Config, logger *slog.Logger) string {
	var handler http.Handler
	if dataSource == nil {
		handler = promhttp.Handler()
	} else {
		// Each request scrapes a broker whose credentials / scrapeURI come from the request itself. We therefore
		// work on a per-request Config copy so that concurrent scrapes can NOT overwrite each other's credentials
		// on a shared Config (which caused broker-wide SEMP 401s).
		reqConf := resolveRequestConfig(r, conf, logger)

		logger.Info("handle http request", "dataSource", logDataSource(dataSource), "scrapeURI", reqConf.ScrapeURI)

		exp := exporter.NewExporter(r.Context(), logger, reqConf, &dataSource)
		registry := prometheus.NewRegistry()
		registry.MustRegister(exp)
		handler = promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	}
	securedHandler := web.WrapWithAuth(handler, conf.ExporterAuth)

	securedHandler.ServeHTTP(w, r)
	return w.Header().Get("status")
}

// parseDataSources builds the list of scrape targets from the request form. Each `m.<Name>` parameter holds
// `vpnFilter|itemFilter[|metricFilter,...]`; entries with fewer than two `|`-separated parts are skipped with a log.
func parseDataSources(form url.Values, logger *slog.Logger) []exporter.DataSource {
	var dataSource []exporter.DataSource
	for key, values := range form {
		if !strings.HasPrefix(key, "m.") {
			continue
		}
		for _, value := range values {
			parts := strings.Split(value, "|")
			if len(parts) < 2 {
				logger.Error("One or two | expected. Use VPN wildcard | Item wildcard | Optional metric filter for v2 apis", "key", key, "value", value)
				continue
			}

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
	return dataSource
}

// resolveRequestConfig returns a per-request copy of conf with the credentials, scrape URI and timeout overridden
// from the request. For each value the form parameter wins, then the x-solace-broker-* header, otherwise the value
// configured on the base Config is kept. The shared base conf is never mutated, so concurrent requests are fully
// isolated from one another.
func resolveRequestConfig(r *http.Request, conf *exporter.Config, logger *slog.Logger) *exporter.Config {
	reqConf := conf.Clone()

	if username := firstNonEmpty(r.FormValue("username"), r.Header.Get("x-solace-broker-username")); username != "" {
		reqConf.Username = username
	}
	if password := firstNonEmpty(r.FormValue("password"), r.Header.Get("x-solace-broker-password")); password != "" {
		reqConf.Password = password
	}
	if scrapeURI := firstNonEmpty(r.FormValue("scrapeURI"), r.Header.Get("x-solace-broker-scrapeuri")); scrapeURI != "" {
		reqConf.ScrapeURI = scrapeURI
	}
	if timeout := firstNonEmpty(r.FormValue("timeout"), r.Header.Get("x-solace-broker-timeout")); timeout != "" {
		parsed, err := time.ParseDuration(timeout)
		switch {
		case err != nil:
			// Keep the configured timeout instead of silently disabling it (an invalid value used to zero it).
			logger.Error("Per HTTP given timeout parameter is not valid", "err", err, "timeout", timeout)
		case parsed <= 0:
			// A non-positive timeout means "no timeout" for http.Client; a hung broker could then pin a scrape
			// (and a SEMP connection slot) forever. Keep the configured timeout instead.
			logger.Error("Per HTTP given timeout must be positive; keeping configured timeout", "timeout", timeout)
		default:
			reqConf.Timeout = parsed
		}
	}

	return reqConf
}

// firstNonEmpty returns the first non-empty string of the given values, or "" if all are empty.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if len(v) > 0 {
			return v
		}
	}
	return ""
}
