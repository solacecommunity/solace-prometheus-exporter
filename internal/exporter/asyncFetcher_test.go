package exporter

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"solace_exporter/internal/semp"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"golang.org/x/sync/semaphore"
)

func TestDeprecateAllAndDeleteDeprecated(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create a dummy Semp to create metrics
	s := semp.NewSemp(logger, "http://localhost:8080", http.Client{}, nil, false, false)
	desc := semp.NewSemDesc("test_metric", "test", "help", []string{"label"})

	metric1 := s.NewMetric(desc, prometheus.GaugeValue, 1.0, "val1")
	metric2 := s.NewMetric(desc, prometheus.GaugeValue, 2.0, "val2")

	fetcher := &AsyncFetcher{
		metrics: make(map[string]semp.PrometheusMetric),
		logger:  logger,
	}

	fetcher.metrics[metric1.Name()] = metric1
	fetcher.metrics[metric2.Name()] = metric2

	if len(fetcher.metrics) != 2 {
		t.Fatalf("Expected 2 metrics, got %d", len(fetcher.metrics))
	}

	// Call DeprecateAll
	fetcher.DeprecateAll()

	// Check if they are deprecated in the map
	for name, m := range fetcher.metrics {
		if !m.IsDeprecated() {
			t.Errorf("Metric %s should be deprecated but is not", name)
		}
	}

	// Call DeleteDeprecated
	fetcher.DeleteDeprecated()

	// Check if they are deleted
	if len(fetcher.metrics) != 0 {
		t.Errorf("Expected 0 metrics after DeleteDeprecated, but got %d. Stale metrics: %v", len(fetcher.metrics), fetcher.metrics)
	}
}

type mockCollector struct {
	m prometheus.Metric
}

func (c *mockCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.m.Desc()
}

func (c *mockCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- c.m
}

func TestNewAsyncFetcher(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// state variable
	state := 0

	// setup a mock webserver
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch state {
		case 0:
			// That return an ok response for: getQueueDetailsSemp1
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<rpc-reply semp-version="soltr/9_1_1VMR"><rpc><show><queue><queues><queue><name>q1</name><info><message-vpn>default</message-vpn></info></queue></queues></queue></show></rpc><execute-result code="ok"/></rpc-reply>`))
		case 1:
			// That return an a 500 after 2sec body="pending shutdown" for: getQueueDetailsSemp1
			time.Sleep(2 * time.Second)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`pending shutdown`))
		case 2:
			// That return an a 500 body="pending shutdown. see log for further details" for: getQueueDetailsSemp1
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`pending shutdown. see log for further details`))
		case 3:
			// That return an ok response for: getQueueDetailsSemp1
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<rpc-reply semp-version="soltr/9_1_1VMR"><rpc><show><queue><queues><queue><name>q1</name><info><message-vpn>default</message-vpn></info></queue></queues></queue></show></rpc><execute-result code="ok"/></rpc-reply>`))
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conf := &Config{
		PrefetchInterval: 50 * time.Millisecond,
		Timeout:          5 * time.Second,
		ScrapeURI:        server.URL,
	}

	ds := []DataSource{{Name: "QueueDetails"}}
	connections := semaphore.NewWeighted(1)

	fetcher := NewAsyncFetcher(ctx, "getQueueDetailsSemp1", ds, conf, logger, connections)

	// state 0: ok
	time.Sleep(100 * time.Millisecond) // Let it fetch
	// expect solace_up to be 1
	// actually it's easier to check fetcher.metrics["solace_up"] or similar
	f := fetcher

	mc := func(m prometheus.Metric) prometheus.Collector {
		return &mockCollector{m}
	}

	checkUp := func(expected float64) {
		time.Sleep(300 * time.Millisecond)
		f.mutex.Lock()
		defer f.mutex.Unlock()
		found := false
		for k, v := range f.metrics {
			if strings.Contains(k, "solace_up") {
				col := mc(v.AsPrometheusMetric())
				val := testutil.ToFloat64(col)
				if val != expected {
					t.Errorf("Expected solace_up to be %f, got %f", expected, val)
				}
				found = true
			}
		}
		if !found && expected == 1 {
			t.Errorf("solace_up metric not found")
		}
	}

	checkUp(1.0)

	state = 1
	time.Sleep(2500 * time.Millisecond) // Wait for the 2sec timeout and fetch
	state = 2
	time.Sleep(500 * time.Millisecond)
	// expect solace_up to be 0
	checkUp(0.0)

	state = 3
	time.Sleep(1000 * time.Millisecond)
	// expect solace_up to be 1
	checkUp(1.0)
}
