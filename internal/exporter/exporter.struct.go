package exporter

import (
	"context"
	"log/slog"
	"solace_exporter/internal/semp"
)

// Exporter collects Solace stats from the given URI and exports them using
// the prometheus metrics package.
type Exporter struct {
	config     *Config
	dataSource *[]DataSource
	logger     *slog.Logger
	semp       *semp.Semp
}

// NewExporter returns an initialized Exporter.
func NewExporter(ctx context.Context, logger *slog.Logger, conf *Config, dataSource *[]DataSource) *Exporter {
	httpVisitor, err := conf.httpVisitor(ctx)
	if err != nil {
		logger.Error("Failed to create HTTP visitor for exporter", "err", err)
	}

	return &Exporter{
		logger:     logger,
		config:     conf,
		dataSource: dataSource,
		semp:       semp.NewSemp(logger, conf.ScrapeURI, conf.newHTTPClient(), httpVisitor, conf.logBrokerToSlowWarnings, conf.IsHWBroker),
	}
}
