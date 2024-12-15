package exporter

import (
	"context"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/semaphore"
	"solace_exporter/semp"
	"sync"
	"time"
)

const (
	// Capacity for the channel to collect metrics and descriptors.
	capMetricChan = 1000
)

func NewAsyncFetcher(urlPath string, dataSource []DataSource, conf Config, logger log.Logger, connections *semaphore.Weighted, version float64) *AsyncFetcher {

	var fetcher = &AsyncFetcher{
		dataSource: dataSource,
		conf:       conf,
		logger:     logger,
		metrics:    make(map[string]semp.PrometheusMetric),
		exporter:   NewExporter(logger, &conf, &dataSource, version),
	}

	collectWorker := func() {
		ctx := context.Background()
		for {
			if err := connections.Acquire(ctx, 1); err != nil {
				_ = level.Error(logger).Log("msg", "Failed to acquire semaphore", "handler", "/"+urlPath, "err", err)
				continue
			}

			_ = level.Debug(logger).Log("msg", "Fetching for handler", "handler", "/"+urlPath)

			var startTime = time.Now()
			readMetrics(fetcher)

			connections.Release(1)

			// _ = level.Debug(logger).Log("msg", "Finished fetching for handler", "handler", "/"+urlPath)
			// Be nice to the broker and wait between scrapes and let other threads fetch data.
			sleepUntilNextIteration(startTime, conf.PrefetchInterval)
		}
	}

	go collectWorker()

	return fetcher
}

func sleepUntilNextIteration(startTime time.Time, interval time.Duration) {
	now := time.Now()
	nextInterval := startTime.Add(interval)
	if nextInterval.After(now) {
		timeToSleep := nextInterval.Sub(now)
		time.Sleep(timeToSleep)
	}
}

type AsyncFetcher struct {
	mutex      sync.Mutex
	dataSource []DataSource
	conf       Config
	logger     log.Logger
	metrics    map[string]semp.PrometheusMetric
	exporter   *Exporter
}

func readMetrics(f *AsyncFetcher) {
	var metricsChan = make(chan semp.PrometheusMetric, capMetricChan)
	var wg sync.WaitGroup
	wg.Add(1)

	f.DeprecateAll()

	collectWorker := func() {
		f.exporter.CollectPrometheusMetric(metricsChan)
		wg.Done()
	}
	go collectWorker()

	go func() {
		wg.Wait()
		close(metricsChan)
	}()

	// read from chanel until the channel is closed
	cache := make([]semp.PrometheusMetric, 0, 100)
	for {
		metric, ok := <-metricsChan
		if !ok {
			break
		}
		cache = append(cache, metric)
		if len(cache) >= 100 {
			// Update cache by chunks of 100 to provide updated metrics as early as possible
			f.Merge(cache)
			cache = make([]semp.PrometheusMetric, 0, 100)
		}
	}

	f.Merge(cache)
	f.DeleteDeprecated()
}

func (f *AsyncFetcher) Describe(descs chan<- *prometheus.Desc) {
	f.exporter.Describe(descs)
}

func (f *AsyncFetcher) Collect(metrics chan<- prometheus.Metric) {
	f.mutex.Lock()
	for _, metric := range f.metrics {
		metrics <- metric.AsPrometheusMetric()
	}
	f.mutex.Unlock()
}

func (f *AsyncFetcher) DeprecateAll() {
	f.mutex.Lock()
	for _, metric := range f.metrics {
		metric.Deprecate()
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
