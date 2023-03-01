package exporter

import (
	"solace_exporter/semp"

	"github.com/go-kit/kit/log"
)

// Exporter collects Solace stats from the given URI and exports them using
// the prometheus metrics package.
type Exporter struct {
	config    *Config
	dataSource  *[]DataSource
	logger    log.Logger
	lastError error
	semp      *semp.Semp
}

// NewExporter returns an initialized Exporter.
func NewExporter(logger log.Logger, conf *Config, dataSource *[]DataSource, version float64) *Exporter {
	return &Exporter{
		logger:    logger,
		config:    conf,
		dataSource: dataSource,
		lastError: nil,
		semp:      semp.NewSemp(logger, conf.ScrapeURI, conf.newHttpClient(), conf.httpVisitor(), version),
	}
}
