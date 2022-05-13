package exporter

import (
	"github.com/go-kit/kit/log"
	"solace_exporter/semp"
)

// Exporter collects Solace stats from the given URI and exports them using
// the prometheus metrics package.
type Exporter struct {
	config    *Config
	logger    log.Logger
	lastError error
	semp      *semp.Semp
}

// NewExporter returns an initialized Exporter.
func NewExporter(logger log.Logger, conf *Config, version float64) *Exporter {
	return &Exporter{
		logger:    logger,
		config:    conf,
		lastError: nil,
		semp:      semp.NewSemp(logger, conf.ScrapeURI, conf.newHttpClient(), conf.httpVisitor(), version),
	}
}
