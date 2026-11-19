package semp

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// newSempReplyingWith returns a Semp wired to a test broker that always answers with reply.
func newSempReplyingWith(t *testing.T, reply string, isHWBroker bool) *Semp {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(server.Close)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	return NewSemp(logger, server.URL, http.Client{}, nil, false, isHWBroker)
}

// metricValuesByFQName indexes values by fully qualified metric name. Only valid where each name is emitted once.
func metricValuesByFQName(metrics []PrometheusMetric) map[string]float64 {
	out := make(map[string]float64, len(metrics))
	for _, m := range metrics {
		out[m.desc.fqName] = m.value
	}
	return out
}

// ---- Health ----

// healthReply uses distinct microsecond values per field so a swapped min/max/avg/cur pairing is detectable.
const healthReply = `<rpc-reply><rpc><show><system><health>` +
	`<disk-latency-minimum-value>1000000</disk-latency-minimum-value>` +
	`<disk-latency-maximum-value>2000000</disk-latency-maximum-value>` +
	`<disk-latency-average-value>3000000</disk-latency-average-value>` +
	`<disk-latency-current-value>4000000</disk-latency-current-value>` +
	`<compute-latency-minimum-value>5000000</compute-latency-minimum-value>` +
	`<compute-latency-maximum-value>6000000</compute-latency-maximum-value>` +
	`<compute-latency-average-value>7000000</compute-latency-average-value>` +
	`<compute-latency-current-value>8000000</compute-latency-current-value>` +
	`<mate-link-latency-minimum-value>9000000</mate-link-latency-minimum-value>` +
	`<mate-link-latency-maximum-value>10000000</mate-link-latency-maximum-value>` +
	`<mate-link-latency-average-value>11000000</mate-link-latency-average-value>` +
	`<mate-link-latency-current-value>12000000</mate-link-latency-current-value>` +
	`</health></system></show></rpc><execute-result code="ok"/></rpc-reply>`

// TestGetHealthSemp1ConvertsMicrosecondsToSeconds pins both the microsecond->second conversion (the metric names end
// in _seconds, dashboards rely on it) and the mapping of each SEMP element to its own metric.
func TestGetHealthSemp1ConvertsMicrosecondsToSeconds(t *testing.T) {
	t.Parallel()
	s := newSempReplyingWith(t, healthReply, false)

	ch := make(chan PrometheusMetric, 50)
	up, err := s.GetHealthSemp1(ch)
	values := metricValuesByFQName(drain(ch))

	if err != nil || up != 1 {
		t.Fatalf("up=%v err=%v, want up=1 and no error", up, err)
	}
	want := map[string]float64{
		"solace_system_disk_latency_min_seconds":      1,
		"solace_system_disk_latency_max_seconds":      2,
		"solace_system_disk_latency_avg_seconds":      3,
		"solace_system_disk_latency_cur_seconds":      4,
		"solace_system_compute_latency_min_seconds":   5,
		"solace_system_compute_latency_max_seconds":   6,
		"solace_system_compute_latency_avg_seconds":   7,
		"solace_system_compute_latency_cur_seconds":   8,
		"solace_system_mate_link_latency_min_seconds": 9,
		"solace_system_mate_link_latency_max_seconds": 10,
		"solace_system_mate_link_latency_avg_seconds": 11,
		"solace_system_mate_link_latency_cur_seconds": 12,
	}
	if len(values) != len(want) {
		t.Errorf("emitted %d metrics, want %d: %v", len(values), len(want), values)
	}
	for name, w := range want {
		if got, ok := values[name]; !ok || got != w {
			t.Errorf("%s = %v (present=%v), want %v", name, got, ok, w)
		}
	}
}

// TestGetHealthSemp1RejectsNotOkResult covers the recoverable-failure branch (e.g. an appliance, where the command
// is unsupported): up=0, an error and no metrics.
func TestGetHealthSemp1RejectsNotOkResult(t *testing.T) {
	t.Parallel()
	s := newSempReplyingWith(t, `<rpc-reply><rpc/><execute-result code="fail" reason="not supported"/></rpc-reply>`, false)

	ch := make(chan PrometheusMetric, 50)
	up, err := s.GetHealthSemp1(ch)
	metrics := drain(ch)

	if err == nil {
		t.Error("expected an error")
	}
	if up != 0 {
		t.Errorf("up = %v, want 0", up)
	}
	if len(metrics) != 0 {
		t.Errorf("emitted %d metrics, want 0", len(metrics))
	}
}

// ---- Redundancy ----

func redundancyReply(role, primaryActivity, backupActivity string) string {
	return `<rpc-reply><rpc><show><redundancy>` +
		`<config-status>Enabled</config-status>` +
		`<redundancy-status>Up</redundancy-status>` +
		`<operating-mode>Message Routing Node</operating-mode>` +
		`<redundancy-mode>Active/Standby</redundancy-mode>` +
		`<active-standby-role>` + role + `</active-standby-role>` +
		`<mate-router-name>mate1</mate-router-name>` +
		`<oper-status><adb-link-up>true</adb-link-up><adb-hello-up>false</adb-hello-up></oper-status>` +
		`<virtual-routers>` +
		`<primary><status><activity>` + primaryActivity + `</activity></status></primary>` +
		`<backup><status><activity>` + backupActivity + `</activity></status></backup>` +
		`</virtual-routers></redundancy></show></rpc><execute-result code="ok"/></rpc-reply>`
}

// TestGetRedundancySemp1LocalActive pins when the node counts as the active messaging node: only when the virtual
// router matching its own role reports "Local Active". Alerts on failover depend on this.
func TestGetRedundancySemp1LocalActive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		role    string
		primary string
		backup  string
		want    float64
	}{
		{"primary node, primary vrouter active", "Primary", "Local Active", "Mate Active", 1},
		{"primary node, primary vrouter standby", "Primary", "Mate Active", "Local Active", 0},
		{"backup node, backup vrouter active", "Backup", "Mate Active", "Local Active", 1},
		{"backup node, backup vrouter standby", "Backup", "Local Active", "Mate Active", 0},
		{"monitor node is never active", "Monitor", "Local Active", "Local Active", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newSempReplyingWith(t, redundancyReply(tt.role, tt.primary, tt.backup), false)

			ch := make(chan PrometheusMetric, 20)
			up, err := s.GetRedundancySemp1(ch)
			values := metricValuesByFQName(drain(ch))

			if err != nil || up != 1 {
				t.Fatalf("up=%v err=%v, want up=1 and no error", up, err)
			}
			if got := values["solace_system_redundancy_local_active"]; got != tt.want {
				t.Errorf("local_active = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetRedundancySemp1EncodesStatesAndGatesHardwareMetrics checks the enum encodings, the mate_name label, and that
// the *_hw_* metrics are emitted only for hardware brokers.
func TestGetRedundancySemp1EncodesStatesAndGatesHardwareMetrics(t *testing.T) {
	t.Parallel()

	reply := redundancyReply("Primary", "Local Active", "Mate Active")
	hwMetrics := []string{
		"solace_system_redundancy_hw_mode",
		"solace_system_redundancy_hw_adb_link",
		"solace_system_redundancy_hw_adb_hello",
	}

	t.Run("software broker emits no hw metrics", func(t *testing.T) {
		t.Parallel()
		s := newSempReplyingWith(t, reply, false)
		ch := make(chan PrometheusMetric, 20)
		if _, err := s.GetRedundancySemp1(ch); err != nil {
			t.Fatalf("error: %v", err)
		}
		metrics := drain(ch)
		values := metricValuesByFQName(metrics)

		for _, name := range hwMetrics {
			if _, ok := values[name]; ok {
				t.Errorf("%s must not be emitted for a software broker", name)
			}
		}
		// Enabled=1 of {Disabled, Enabled, Shutdown}; Up=1; Primary=1 of {Backup, Primary, Monitor, Undefined}.
		for name, want := range map[string]float64{
			"solace_system_redundancy_config": 1,
			"solace_system_redundancy_up":     1,
			"solace_system_redundancy_role":   1,
		} {
			if got := values[name]; got != want {
				t.Errorf("%s = %v, want %v", name, got, want)
			}
		}
		for _, m := range metrics {
			if len(m.labelValues) != 1 || m.labelValues[0] != "mate1" {
				t.Errorf("%s labels = %v, want [mate1]", m.desc.fqName, m.labelValues)
			}
		}
	})

	t.Run("hardware broker emits hw metrics", func(t *testing.T) {
		t.Parallel()
		s := newSempReplyingWith(t, reply, true)
		ch := make(chan PrometheusMetric, 20)
		if _, err := s.GetRedundancySemp1(ch); err != nil {
			t.Fatalf("error: %v", err)
		}
		values := metricValuesByFQName(drain(ch))

		// Active/Standby=1; adb-link-up true=1; adb-hello-up false=0.
		for name, want := range map[string]float64{
			"solace_system_redundancy_hw_mode":      1,
			"solace_system_redundancy_hw_adb_link":  1,
			"solace_system_redundancy_hw_adb_hello": 0,
		} {
			got, ok := values[name]
			if !ok || got != want {
				t.Errorf("%s = %v (present=%v), want %v", name, got, ok, want)
			}
		}
	})
}

// TestGetRedundancySemp1RejectsNotOkResult covers the recoverable-failure branch.
func TestGetRedundancySemp1RejectsNotOkResult(t *testing.T) {
	t.Parallel()
	s := newSempReplyingWith(t, `<rpc-reply><rpc/><execute-result code="fail" reason="nope"/></rpc-reply>`, false)

	ch := make(chan PrometheusMetric, 20)
	up, err := s.GetRedundancySemp1(ch)
	metrics := drain(ch)

	if err == nil {
		t.Error("expected an error")
	}
	if up != 0 {
		t.Errorf("up = %v, want 0", up)
	}
	if len(metrics) != 0 {
		t.Errorf("emitted %d metrics, want 0", len(metrics))
	}
}
