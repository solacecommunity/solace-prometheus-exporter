package exporter

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"sync"
	"time"
)

const (
	// Capacity for the channel to collect metrics and descriptors.
	capMetricChan = 1000
)

func NewAsyncFetcher(urlPath string, dataSource []DataSource, conf Config, logger log.Logger, version float64) *AsyncFetcher {

	var fetcher = &AsyncFetcher{
		dataSource: dataSource,
		conf:       conf,
		logger:     logger,
		metrics:    make([]prometheus.Metric, 0, capMetricChan),
		exporter:   NewExporter(logger, &conf, &dataSource, version),
	}

	collectWorker := func() {
		for {
			_ = level.Debug(logger).Log("msg", "Fetching for handler", "handler", "/"+urlPath)

			readMetrics(fetcher)

			// _ = level.Debug(logger).Log("msg", "Finished fetching for handler", "handler", "/"+urlPath)
			time.Sleep(conf.PrefetchInterval)
		}
	}

	go collectWorker()

	return fetcher
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
