// Package version carries the build metadata of the exporter binary and exposes it as a Prometheus metric.
//
// The three variables below are deliberately plain package-level strings so the release pipeline can inject them
// at link time:
//
//	go build -ldflags "-X solace_exporter/internal/version.Version=1.4.2 \
//	                   -X solace_exporter/internal/version.Commit=abc1234 \
//	                   -X solace_exporter/internal/version.BuildDate=2026-08-01" ./cmd/solace-prometheus-exporter
//
// A plain `go build` leaves them at their fallback values, which is what keeps a locally built binary
// distinguishable from a released image instead of silently claiming a version it does not have.
package version

import "github.com/prometheus/client_golang/prometheus"

// Fallbacks used when the linker did not inject a value (local build, `go run`, `go test`).
const (
	unknownVersion   = "dev"
	unknownCommit    = "unknown"
	unknownBuildDate = "unknown"
)

var (
	// Version is the released version of the build, derived from the git tag (e.g. "1.4.2").
	Version = unknownVersion
	// Commit is the short git commit SHA the build was cut from (e.g. "abc1234").
	Commit = unknownCommit
	// BuildDate is the UTC build date in RFC 3339 date form (e.g. "2026-08-01").
	BuildDate = unknownBuildDate
)

// buildInfoName is the fully qualified name of the build info metric. It is deliberately not built through
// semp.NewSemDesc: that registry describes broker metrics, whereas this one describes the exporter process.
//
// Note that this is the same name github.com/prometheus/common/version.NewCollector("solace_exporter") would
// produce. Registering that collector as well would panic on a duplicate metric name, so we own this name here
// and use prometheus/common/version only for the "Build context" startup log line.
const buildInfoName = "solace_exporter_build_info"

// collector emits a single constant-1 gauge whose labels carry the build metadata. This is the standard
// Prometheus "info metric" pattern: the value is meaningless on its own and exists only so the labels can be
// joined onto other series, e.g. `solace_up * on(instance) group_left(version) solace_exporter_build_info`.
type collector struct {
	desc *prometheus.Desc
}

// NewCollector returns a prometheus.Collector exposing solace_exporter_build_info. The label values are snapshot
// at construction time; they are immutable for the lifetime of the process.
func NewCollector() prometheus.Collector {
	return &collector{
		desc: prometheus.NewDesc(
			buildInfoName,
			"Build information of the running exporter. The value is always 1; the build metadata is carried in the labels.",
			nil,
			prometheus.Labels{
				"version":    Version,
				"commit":     Commit,
				"build_date": BuildDate,
			},
		),
	}
}

// Describe implements prometheus.Collector.
func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

// Collect implements prometheus.Collector.
func (c *collector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, 1)
}
