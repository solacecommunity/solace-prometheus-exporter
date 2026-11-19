package semp

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

const versionFixture = "../../test/data/semp1-version.txt"

func versionReply(currentLoad string, totalSecs string, executeResult string) string {
	return `<rpc-reply semp-version="soltr/9_3VMR"><rpc><show><version>` +
		`<current-load>` + currentLoad + `</current-load>` +
		`<uptime><days>1</days><hours>2</hours><mins>3</mins><secs>4</secs>` +
		`<total-secs>` + totalSecs + `</total-secs></uptime>` +
		`</version></show></rpc>` + executeResult + `</rpc-reply>`
}

func newVersionTestSemp(t *testing.T, handler http.HandlerFunc) *Semp {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	return NewSemp(logger, server.URL, http.Client{}, nil, false, false)
}

func replyWith(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}
}

// valuesByMetricName indexes emitted metric values by fully qualified name (no labels expected for Version).
func valuesByMetricName(metrics []PrometheusMetric) map[string]float64 {
	out := make(map[string]float64, len(metrics))
	for _, m := range metrics {
		out[m.desc.fqName] = m.value
	}
	return out
}

// TestGetVersionSemp1EncodesLoadAsZeroPaddedNumber pins the documented WWWXXXYYYZZZ encoding. Existing alert rules
// compare against these numbers, so a change here would be a breaking change for users.
func TestGetVersionSemp1EncodesLoadAsZeroPaddedNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		currentLoad string
		want        float64
	}{
		{"four-part load with soltr_ prefix", "soltr_9.3.1.8", 9003001008},
		{"double-digit components", "soltr_10.25.1.7", 10025001007},
		{"prefix is optional", "9.1.1.5", 9001001005},
		{"large component", "soltr_10.4.100.12", 10004100012},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newVersionTestSemp(t, replyWith(versionReply(tt.currentLoad, "42", `<execute-result code="ok"/>`)))

			ch := make(chan PrometheusMetric, 10)
			up, err := s.GetVersionSemp1(ch)
			values := valuesByMetricName(drain(ch))

			if err != nil {
				t.Fatalf("GetVersionSemp1 error: %v", err)
			}
			if up != 1 {
				t.Errorf("up = %v, want 1", up)
			}
			if got := values["solace_system_version_currentload"]; got != tt.want {
				t.Errorf("version number = %v, want %v", got, tt.want)
			}
			if got := values["solace_system_version_uptime_totalsecs"]; got != 42 {
				t.Errorf("uptime total secs = %v, want 42", got)
			}
		})
	}
}

// TestGetVersionSemp1RejectsNotOkResult covers the recoverable-failure branch: up=0, an error, and no metrics.
func TestGetVersionSemp1RejectsNotOkResult(t *testing.T) {
	t.Parallel()
	s := newVersionTestSemp(t, replyWith(versionReply("soltr_9.3.1.8", "42", `<execute-result code="fail" reason="denied"/>`)))

	ch := make(chan PrometheusMetric, 10)
	up, err := s.GetVersionSemp1(ch)
	metrics := drain(ch)

	if err == nil || !strings.Contains(err.Error(), "denied") {
		t.Errorf("expected an error carrying the broker reason, got %v", err)
	}
	if up != 0 {
		t.Errorf("up = %v, want 0", up)
	}
	if len(metrics) != 0 {
		t.Errorf("emitted %d metrics on a failed command, want 0", len(metrics))
	}
}

// TestGetVersionSemp1MalformedXMLIsRecoverable ensures a garbled reply neither panics nor aborts other targets.
func TestGetVersionSemp1MalformedXMLIsRecoverable(t *testing.T) {
	t.Parallel()
	s := newVersionTestSemp(t, replyWith(`<rpc-reply><rpc><show><version>`))

	ch := make(chan PrometheusMetric, 10)
	up, err := s.GetVersionSemp1(ch)
	metrics := drain(ch)

	if err == nil {
		t.Error("expected a decode error")
	}
	if up != 0 {
		t.Errorf("up = %v, want 0 (recoverable, not -1)", up)
	}
	if len(metrics) != 0 {
		t.Errorf("emitted %d metrics, want 0", len(metrics))
	}
}

// TestGetVersionSemp1UnreachableBrokerIsUnrecoverable covers the up=-1 contract that aborts the remaining targets.
func TestGetVersionSemp1UnreachableBrokerIsUnrecoverable(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.NotFoundHandler())
	s := NewSemp(slog.New(slog.NewTextHandler(os.Stdout, nil)), server.URL, http.Client{}, nil, false, false)
	server.Close() // nothing listens on the address any more

	ch := make(chan PrometheusMetric, 10)
	up, err := s.GetVersionSemp1(ch)
	metrics := drain(ch)

	if err == nil {
		t.Error("expected a connection error")
	}
	if up != -1 {
		t.Errorf("up = %v, want -1", up)
	}
	if len(metrics) != 0 {
		t.Errorf("emitted %d metrics, want 0", len(metrics))
	}
}

// TestGetVersionSemp1AgainstCapturedBrokerPayload runs the getter on a real broker capture. The fixture starts with
// a curl command line, so everything before the XML declaration or root element is stripped.
func TestGetVersionSemp1AgainstCapturedBrokerPayload(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(versionFixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	payload := string(raw)
	start := strings.Index(payload, "<rpc-reply")
	if start < 0 {
		t.Fatal("fixture has no <rpc-reply> element")
	}
	s := newVersionTestSemp(t, replyWith(payload[start:]))

	ch := make(chan PrometheusMetric, 10)
	up, err := s.GetVersionSemp1(ch)
	values := valuesByMetricName(drain(ch))

	if err != nil || up != 1 {
		t.Fatalf("up=%v err=%v, want up=1 and no error", up, err)
	}
	if got := values["solace_system_version_currentload"]; got != 9003001008 {
		t.Errorf("version number = %v, want 9003001008 (soltr_9.3.1.8)", got)
	}
	if got := values["solace_system_version_uptime_totalsecs"]; got != 3738 {
		t.Errorf("uptime total secs = %v, want 3738", got)
	}
}
