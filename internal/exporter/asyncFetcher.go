package exporter

import (
	"context"
	"log/slog"
	"solace_exporter/internal/semp"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/semaphore"
)

const (
	// Capacity for the channel to collect metrics and descriptors.
	capMetricChan = 1000

	// Number of metrics to cache before merging into the main map.
	metricCacheChunkSize = 100
)

func NewAsyncFetcher(ctx context.Context, urlPath string, dataSource []DataSource, conf *Config, logger *slog.Logger, connections *semaphore.Weighted) *AsyncFetcher {
	var fetcher = &AsyncFetcher{
		dataSource: dataSource,
		conf:       conf,
		logger:     logger,
		metrics:    make(map[string]semp.PrometheusMetric),
		exporter:   NewExporter(ctx, logger, conf, &dataSource),
	}

	collectWorker := func() {
		ticker := time.NewTicker(conf.PrefetchInterval)
		defer ticker.Stop()

		for {
			if err := connections.Acquire(ctx, 1); err != nil {
				logger.Error("Failed to acquire semaphore", "handler", "/"+urlPath, "err", err)

				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Second):
					continue
				}
			}

			logger.Debug("Fetching for handler", "handler", "/"+urlPath)

			readMetrics(fetcher)

			connections.Release(1)

			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				continue
			}
		}
	}

	go collectWorker()

	return fetcher
}

type AsyncFetcher struct {
	mutex      sync.Mutex
	dataSource []DataSource
	conf       *Config
	logger     *slog.Logger
	metrics    map[string]semp.PrometheusMetric
	exporter   *Exporter
}

func readMetrics(f *AsyncFetcher) {
	var metricsChan = make(chan semp.PrometheusMetric, capMetricChan)

	f.DeprecateAll()

	go func() {
		defer close(metricsChan)
		f.exporter.CollectPrometheusMetric(metricsChan)
	}()

	// read from channel until the channel is closed
	cache := make([]semp.PrometheusMetric, 0, metricCacheChunkSize)
	for metric := range metricsChan {
		cache = append(cache, metric)
		if len(cache) >= metricCacheChunkSize {
			// Update cache by chunks to provide updated metrics as early as possible
			f.Merge(cache)
			cache = make([]semp.PrometheusMetric, 0, metricCacheChunkSize)
		}
	}

	f.Merge(cache)
	f.DeleteDeprecated()
}

func (f *AsyncFetcher) Describe(desc chan<- *prometheus.Desc) {
	f.exporter.Describe(desc)
}

func (f *AsyncFetcher) Collect(metrics chan<- prometheus.Metric) {
	f.mutex.Lock()
	copiedMetrics := make([]prometheus.Metric, 0, len(f.metrics))
	for _, metric := range f.metrics {
		copiedMetrics = append(copiedMetrics, metric.AsPrometheusMetric())
	}
	f.mutex.Unlock()

	for _, metric := range copiedMetrics {
		metrics <- metric
	}
}

func (f *AsyncFetcher) DeprecateAll() {
	f.mutex.Lock()
	for key, metric := range f.metrics {
		metric.Deprecate()
		f.metrics[key] = metric
	}
	f.mutex.Unlock()
}

func (f *AsyncFetcher) Merge(cache []semp.PrometheusMetric) {
	f.mutex.Lock()
	for _, metric := range cache {
		f.metrics[metric.Name()] = metric
	}
	f.mutex.Unlock()
}

func (f *AsyncFetcher) DeleteDeprecated() {
	f.mutex.Lock()
	for key, metric := range f.metrics {
		if metric.IsDeprecated() {
			delete(f.metrics, key)
		}
	}
	f.mutex.Unlock()
}
