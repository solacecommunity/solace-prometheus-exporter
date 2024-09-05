package exporter

import (
	"github.com/prometheus/client_golang/prometheus"
	"solace_exporter/semp"
)

// Describe describes all the metrics ever exported by the Solace exporter. It
// implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, metricDescItems := range semp.MetricDesc {
		for _, m := range metricDescItems {
			ch <- m.AsPrometheusDesc()
		}
	}
}
