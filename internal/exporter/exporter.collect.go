package exporter

import (
	"errors"
	"solace_exporter/internal/semp"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// CollectPrometheusMetric fetches the stats of every configured data source from the broker and delivers them as
// metrics, followed by one solace_up metric per data source. An unrecoverable failure (up < 0, e.g. the broker is
// unreachable) is reported once under endpoint "global" and skips the remaining data sources.
func (e *Exporter) CollectPrometheusMetric(ch chan<- semp.PrometheusMetric) {
	for _, dataSource := range *e.dataSource {
		up, err := e.scrape(ch, dataSource)
		if !e.reportUp(ch, dataSource.Name, up, err) {
			return
		}
	}
}

// scrape collects one data source, rejecting unknown names and targets the configured broker type cannot serve.
func (e *Exporter) scrape(ch chan<- semp.PrometheusMetric, dataSource DataSource) (float64, error) {
	target, ok := scrapeTargets[dataSource.Name]
	if !ok {
		return 0, e.targetError("Unknown", dataSource.Name)
	}

	switch {
	case target.brokerKind == softwareBrokerOnly && e.config.IsHWBroker:
		return 0, e.targetError("Software only", dataSource.Name)
	case target.brokerKind == hardwareBrokerOnly && !e.config.IsHWBroker:
		return 0, e.targetError("Hardware only", dataSource.Name)
	}

	return target.scrape(e, ch, dataSource)
}

// targetError logs and returns the error for a data source that cannot be scraped; kind prefixes the message.
func (e *Exporter) targetError(kind string, name string) error {
	msg := kind + " scrape target: \"" + name + "\". Please check documentation for valid targets."
	e.logger.Error(msg)
	return errors.New(msg)
}

// reportUp emits the solace_up metric for one data source and reports whether the remaining data sources should
// still be scraped.
func (e *Exporter) reportUp(ch chan<- semp.PrometheusMetric, endpoint string, up float64, err error) bool {
	if up >= 1 {
		ch <- e.semp.NewMetric(semp.MetricDesc["Global"]["up"], prometheus.GaugeValue, 1, "", endpoint)
		return true
	}

	unrecoverable := up < 0
	if unrecoverable {
		// The failure would repeat on every data source, so it is reported once for the whole broker.
		endpoint = "global"
	}

	errorLabel := "Unknown"
	if err != nil {
		errorLabel = err.Error()
	}
	ch <- e.semp.NewMetric(semp.MetricDesc["Global"]["up"], prometheus.GaugeValue, 0, errorLabel, endpoint)

	return !unrecoverable
}

func (e *Exporter) Collect(pch chan<- prometheus.Metric) {
	var ch = make(chan semp.PrometheusMetric, capMetricChan)
	var wg sync.WaitGroup

	wg.Add(1)

	collectWorker := func() {
		defer wg.Done()
		// A malformed/unexpected broker reply must not crash the whole exporter (all brokers). Recover, log, and
		// let this scrape simply report fewer metrics.
		defer func() {
			if r := recover(); r != nil {
				e.logger.Error("recovered from panic while scraping broker", "panic", r, "scrapeURI", e.config.ScrapeURI)
			}
		}()
		e.CollectPrometheusMetric(ch)
	}
	go collectWorker()

	go func() {
		wg.Wait()
		close(ch)
	}()

	// Ensure `checkedMetricChan` and `uncheckedMetricChan` are drained in case of an early return.
	defer func() {
		if ch != nil {
			for range ch {
			}
		}
	}()

	// read from chanel until the channel is closed
	var distinctMetrics = make(map[string]semp.PrometheusMetric)

	for {
		metric, ok := <-ch
		if !ok {
			for _, metric := range distinctMetrics {
				pch <- metric.AsPrometheusMetric()
			}

			return
		}
		// using a map to filter duplicates and use always most current received value
		distinctMetrics[metric.Name()] = metric
	}
}

func (e *Exporter) getVpnName(vpnFilter string) (string, error) {
	if vpnFilter == "*" {
		if len(strings.TrimSpace(e.config.DefaultVpn)) == 0 {
			return "", errors.New("can't scrape Semp2 As vpnFilter was an * given and the defaultVpn is not set in configuration")
		}
		return e.config.DefaultVpn, nil
	}

	return vpnFilter, nil
}
