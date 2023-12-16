// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"golang.org/x/sync/semaphore"
	"net/http"
	"os"
	"solace_exporter/exporter"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	promVersion "github.com/prometheus/common/version"
)

const version = float64(1004005)

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

	kingpin.Parse()

	logger := promlog.New(&promlogConfig)

	endpoints, conf, err := exporter.ParseConfig(*configFile)
	if err != nil {
		level.Error(logger).Log("msg", err)
		os.Exit(1)
	}

	level.Info(logger).Log("msg", "Starting solace_prometheus_exporter", "version", version)
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
			var asyncFetcher = exporter.NewAsyncFetcher(urlPath, dataSource, *conf, logger, sempConnections, version)
			http.HandleFunc("/"+urlPath, func(w http.ResponseWriter, r *http.Request) {
				doHandleAsync(w, r, asyncFetcher, *conf, logger)
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

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var endpointsDoc bytes.Buffer
		for urlPath, dataSources := range endpoints {
			endpointsDoc.WriteString("<li><a href='/" + urlPath + "'>Custom Exporter " + urlPath + " -> " + logDataSource(dataSources) + "</a></li>")
		}

		w.Write([]byte(`<html>
            <head><title>Solace Exporter</title></head>
            <body>
            <h1>Solace Exporter</h1>
            <ul style="list-style: none;">
                <li><a href='` + "/metrics" + `'>Exporter Metrics</a></li>
				` + endpointsDoc.String() + `
				<li><a href='` + "/solace?m.ClientStats=*|*&m.VpnStats=*|*&m.BridgeStats=*|*&m.QueueRates=*|*" + `'>Solace Broker</a>
				<br>
				<p>Configure the data you want ot receive, via HTTP GET parameters.
				<br>Please use in format &quot;m.ClientStats=*|*&m.VpnStats=*|*&quot; 
				<br>Here is &quot;m.&quot; the prefix.
				<br>Here is &quot;ClientStats&quot; the scrape target.
				<br>The first asterisk the VPN filter and the second asterisk the item filter.
				Not all scrape targets support filter.
				<br>Scrape targets:<br>
				<table><tr><th>scape target</th><th>vpn filter supports</th><th>item filter supported</th><th>performance</th><tr>
					<tr><td>Version</td><td>no</td><td>no</td><td>dont harm broker</td></tr>
					<tr><td>Health</td><td>no</td><td>no</td><td>dont harm broker</td></tr>
					<tr><td>StorageElement</td><td>no</td><td>no</td><td>dont harm broker</td></tr>
					<tr><td>Disk</td><td>no</td><td>yes</td><td>dont harm broker</td></tr>
					<tr><td>Memory</td><td>no</td><td>no</td><td>dont harm broker</td></tr>
					<tr><td>Interface</td><td>no</td><td>yes</td><td>dont harm broker</td></tr>
					<tr><td>GlobalStats</td><td>no</td><td>no</td><td>dont harm broker</td></tr>
					<tr><td>Spool</td><td>no</td><td>no</td><td>dont harm broker</td></tr>
					<tr><td>Redundancy (only for HA broker)</td><td>no</td><td>no</td><td>dont harm broker</td></tr>
					<tr><td>ConfigSyncRouter (only for HA broker)</td><td>no</td><td>no</td><td>dont harm broker</td></tr>
					<tr><td>ReplicationStats (only for DR replication broker)</td><td>no</td><td>no</td><td>dont harm broker</td></tr>
					<tr><td>Vpn</td><td>yes</td><td>no</td><td>dont harm broker</td></tr>
					<tr><td>VpnReplication</td><td>yes</td><td>no</td><td>dont harm broker</td></tr>
					<tr><td>ConfigSyncVpn (only for HA broker)</td><td>yes</td><td>no</td><td>dont harm broker</td></tr>
					<tr><td>Bridge</td><td>yes</td><td>yes</td><td>dont harm broker</td></tr>
					<tr><td>VpnSpool</td><td>yes</td><td>no</td><td>dont harm broker</td></tr>
					<tr><td>Client</td><td>yes</td><td>yes</td><td>may harm broker if many clients</td></tr>
					<tr><td>ClientSlowSubscriber</td><td>yes</td><td>yes</td><td>may harm broker if many clients</td></tr>
					<tr><td>ClientStats</td><td>yes</td><td>no</td><td>may harm broker if many clients</td></tr>
					<tr><td>ClientConnections</td><td>yes</td><td>no</td><td>may harm broker if many clients</td></tr>
					<tr><td>ClientMessageSpoolStats</td><td>yes</td><td>yes</td><td>no</td></tr>
					<tr><td>ClusterLinks</td><td>yes</td><td>no</td><td>may harm broker if many clients</td></tr>
					<tr><td>VpnStats</td><td>yes</td><td>no</td><td>has a very small performance down site</td></tr>
					<tr><td>BridgeStats</td><td>yes</td><td>yes</td><td>has a very small performance down site</td></tr>
					<tr><td>QueueRates</td><td>yes</td><td>yes</td><td>DEPRECATED: may harm broker if many queues</td></tr>
					<tr><td>QueueStats</td><td>yes</td><td>yes</td><td>may harm broker if many queues</td></tr>
					<tr><td>QueueStatsV2</td><td>yes</td><td>yes</td><td>may harm broker if many queues</td></tr>
					<tr><td>QueueDetails</td><td>yes</td><td>yes</td><td>may harm broker if many queues</td></tr>
					<tr><td>TopicEndpointRates</td><td>yes</td><td>yes</td><td>DEPRECATED: may harm broker if many topic-endpoints</td></tr>
					<tr><td>TopicEndpointStats</td><td>yes</td><td>yes</td><td>may harm broker if many topic-endpoints</td></tr>
					<tr><td>TopicEndpointDetails</td><td>yes</td><td>yes</td><td>may harm broker if many topic-endpoints</td></tr>
				</table>
				<br>
				</p>
				</li>
            <ul>
            </body>
            </html>`))
	})

	// start server
	if conf.EnableTLS {
		if err := http.ListenAndServeTLS(conf.ListenAddr, conf.Certificate, conf.PrivateKey, nil); err != nil {
			level.Error(logger).Log("msg", "Error starting HTTP server", "err", err)
			os.Exit(2)
		}
	} else {
		if err := http.ListenAndServe(conf.ListenAddr, nil); err != nil {
			level.Error(logger).Log("msg", "Error starting HTTP server", "err", err)
			os.Exit(2)
		}
	}

}

func doHandleAsync(w http.ResponseWriter, r *http.Request, asyncFetcher *exporter.AsyncFetcher, conf exporter.Config, logger log.Logger) (resultCode string) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(asyncFetcher)
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	handler.ServeHTTP(w, r)

	return w.Header().Get("status")
}

func doHandle(w http.ResponseWriter, r *http.Request, dataSource []exporter.DataSource, conf exporter.Config, logger log.Logger) (resultCode string) {
	if dataSource == nil {
		handler := promhttp.Handler()
		handler.ServeHTTP(w, r)
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

		exp := exporter.NewExporter(logger, &conf, &dataSource, version)
		registry := prometheus.NewRegistry()
		registry.MustRegister(exp)
		handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		handler.ServeHTTP(w, r)
	}
	return w.Header().Get("status")
}
