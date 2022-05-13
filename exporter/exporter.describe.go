package exporter

import (
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"solace_exporter/semp"
	"strings"
)

// Describe describes all the metrics ever exported by the Solace exporter. It
// implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, dataSource := range e.config.DataSource {
		if metricDescItems, ok := semp.MetricDesc[dataSource.Name]; ok {
			for _, m := range metricDescItems {
				ch <- m
			}
		} else {
			permittedNames := make([]string, 0, len(semp.MetricDesc))
			for index := range semp.MetricDesc {
				permittedNames = append(permittedNames, index)
			}
			_ = level.Error(e.logger).Log("msg", "Unexpected data source name: "+dataSource.Name, "permitted", strings.Join(permittedNames, ","))
		}

	}
}
