package semp

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func memoryReply(slotInfos string) string {
	return `<rpc-reply semp-version="soltr/9_1_1VMR"><rpc><show><memory>` +
		`<physical-memory><memory-info><type>Memory</type><total-in-kb>100</total-in-kb>` +
		`<used-in-kb>40</used-in-kb><free-in-kb>60</free-in-kb><buffers-in-kb>1</buffers-in-kb>` +
		`<cached-in-kb>2</cached-in-kb></memory-info></physical-memory>` +
		`<physical-memory-usage-percent>40</physical-memory-usage-percent>` +
		`<subscription-memory-usage-percent>10</subscription-memory-usage-percent>` +
		slotInfos +
		`</memory></show></rpc><execute-result code="ok"/></rpc-reply>`
}

func newMemoryTestSemp(t *testing.T, reply string) *Semp {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(server.Close)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	return NewSemp(logger, server.URL, http.Client{}, nil, false, false)
}

func drain(ch chan PrometheusMetric) []PrometheusMetric {
	close(ch)
	var out []PrometheusMetric
	for m := range ch {
		out = append(out, m)
	}
	return out
}

// TestGetMemorySemp1EmptySlotInfoNoPanic guards the fix for the index-out-of-range panic on brokers that report no
// slot-info (software/cloud brokers). Such a panic in the detached scrape goroutine crashed the whole exporter.
func TestGetMemorySemp1EmptySlotInfoNoPanic(t *testing.T) {
	t.Parallel()
	s := newMemoryTestSemp(t, memoryReply(`<slot-infos></slot-infos>`))

	ch := make(chan PrometheusMetric, 100)
	up, err := s.GetMemorySemp1(ch) // must not panic
	metrics := drain(ch)

	if err != nil {
		t.Fatalf("GetMemorySemp1 error: %v", err)
	}
	if up != 1 {
		t.Errorf("up = %v, want 1", up)
	}
	if len(metrics) == 0 {
		t.Error("expected physical-memory metrics to be emitted even without slot-info")
	}
	for _, m := range metrics {
		if strings.Contains(m.Name(), "nab_buffer_load_factor") {
			t.Errorf("nab_buffer_load_factor must be skipped when slot-info is empty, got %s", m.Name())
		}
	}
}

// TestGetMemorySemp1WithSlotInfo verifies the nab-buffer-load-factor metric is emitted when slot-info is present.
func TestGetMemorySemp1WithSlotInfo(t *testing.T) {
	t.Parallel()
	s := newMemoryTestSemp(t, memoryReply(`<slot-infos><slot-info><slot>1</slot><nab-buffer-load-factor>0.5</nab-buffer-load-factor></slot-info></slot-infos>`))

	ch := make(chan PrometheusMetric, 100)
	up, err := s.GetMemorySemp1(ch)
	metrics := drain(ch)

	if err != nil {
		t.Fatalf("GetMemorySemp1 error: %v", err)
	}
	if up != 1 {
		t.Errorf("up = %v, want 1", up)
	}
	found := false
	for _, m := range metrics {
		if strings.Contains(m.Name(), "nab_buffer_load_factor") {
			found = true
		}
	}
	if !found {
		t.Error("expected nab_buffer_load_factor metric when slot-info is present")
	}
}
