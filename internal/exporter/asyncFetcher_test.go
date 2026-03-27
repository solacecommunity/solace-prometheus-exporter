package exporter

import (
	"log/slog"
	"net/http"
	"os"
	"solace_exporter/internal/semp"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestDeprecateAllAndDeleteDeprecated(t *testing.T) {
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
