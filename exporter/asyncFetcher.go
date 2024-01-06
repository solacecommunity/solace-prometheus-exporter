package exporter

import (
	"context"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/semaphore"
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
		metrics:    make([]prometheus.Metric, 0, capMetricChan),
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
	metrics    []prometheus.Metric
	exporter   *Exporter
}

func readMetrics(f *AsyncFetcher) {
	var metricsChan = make(chan prometheus.Metric, capMetricChan)
	var wg sync.WaitGroup
	wg.Add(1)

	collectWorker := func() {
		f.exporter.Collect(metricsChan)
		wg.Done()
	}
	go collectWorker()

	go func() {
		wg.Wait()
		close(metricsChan)
	}()

	// Drain checkedMetricChan and uncheckedMetricChan in case of premature return.
	defer func() {
		if metricsChan != nil {
			for range metricsChan {
			}
		}
	}()

	// read from chanel until the channel is closed
	metrics := make([]prometheus.Metric, 0, capMetricChan)
	for {
		metric, ok := <-metricsChan
		if !ok {
			break
		}
		metrics = append(metrics, metric)
	}

	f.mutex.Lock()
	f.metrics = metrics
	f.mutex.Unlock()
}

func (f *AsyncFetcher) Describe(descs chan<- *prometheus.Desc) {
	f.exporter.Describe(descs)
}

func (f *AsyncFetcher) Collect(metrics chan<- prometheus.Metric) {
	f.mutex.Lock()
	for _, metric := range f.metrics {
		metrics <- metric
	}
	f.mutex.Unlock()
}
