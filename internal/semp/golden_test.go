package semp

// Characterization ("golden") tests for every SEMP getter.
//
// Each getter is run against a fake broker for a fixed set of scenarios (happy path, paged reply, HTTP error,
// malformed body, failed execute-result). The SEMP requests it sends, its (up, err) return value and every metric it
// emits are serialized into testdata/golden/<Getter>.golden. The tests fail on any deviation, so a refactoring of
// the SEMP layer cannot silently change the exporter's observable behaviour.
//
// The SEMP v1 replies in testdata/semp1 are synthetic: they were derived from the xml struct tags of each getter's
// Data type, every leaf holding a unique number so each emitted value can be traced back to one field. Struct slices
// hold three elements, the third repeating the second, to exercise the consecutive-duplicate skipping on paged replies.
//
// Regenerate the golden files after an intended behaviour change with:
//
//	go test ./internal/semp -run TestGolden -update

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

var updateGolden = flag.Bool("update", false, "rewrite the golden files in testdata/golden")

const (
	goldenVpnFilter  = "vpn*"
	goldenItemFilter = "item*"
	goldenPageSize   = int64(7)

	// page2Marker is the more-cookie RPC returned on page 1; a request containing it is answered with page 2.
	page2Marker = "<rpc><show><golden-page-2/></show></rpc>"
)

type getterCall func(s *Semp, ch chan<- PrometheusMetric) (float64, error)

// semp1Getters maps each fixture name (testdata/semp1/<name>.xml) to a call with fixed filter arguments.
var semp1Getters = map[string]getterCall{
	"GetAlarmSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) { return s.GetAlarmSemp1(ch) },
	"GetBridgeClientCertSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetBridgeClientCertSemp1(ch, goldenVpnFilter, goldenItemFilter, goldenPageSize)
	},
	"GetBridgeDetailSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetBridgeDetailSemp1(ch, goldenVpnFilter, goldenItemFilter, goldenPageSize)
	},
	"GetBridgeRemoteSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetBridgeRemoteSemp1(ch, goldenVpnFilter, goldenItemFilter)
	},
	"GetBridgeSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetBridgeSemp1(ch, goldenVpnFilter, goldenItemFilter, goldenPageSize)
	},
	"GetBridgeStatsSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetBridgeStatsSemp1(ch, goldenVpnFilter, goldenItemFilter, goldenPageSize)
	},
	"GetClientConnectionStatsSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetClientConnectionStatsSemp1(ch, goldenItemFilter)
	},
	"GetClientMessageSpoolEgressSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetClientMessageSpoolEgressSemp1(ch, goldenItemFilter)
	},
	"GetClientMessageSpoolStatsSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetClientMessageSpoolStatsSemp1(ch, goldenItemFilter, goldenPageSize)
	},
	"GetClientProfileSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetClientProfileSemp1(ch, goldenVpnFilter)
	},
	"GetClientSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetClientSemp1(ch, goldenVpnFilter, goldenItemFilter)
	},
	"GetClientSlowSubscriberSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetClientSlowSubscriberSemp1(ch, goldenVpnFilter, goldenItemFilter)
	},
	"GetClientStatsSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetClientStatsSemp1(ch, goldenItemFilter, goldenPageSize)
	},
	"GetClockDetailSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) { return s.GetClockDetailSemp1(ch) },
	"GetClusterLinksSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetClusterLinksSemp1(ch, goldenVpnFilter, goldenItemFilter)
	},
	"GetConfigSyncRouterSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetConfigSyncRouterSemp1(ch)
	},
	"GetConfigSyncSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) { return s.GetConfigSyncSemp1(ch) },
	"GetConfigSyncVpnSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetConfigSyncVpnSemp1(ch, goldenVpnFilter, goldenPageSize)
	},
	"GetDiskSemp1":             func(s *Semp, ch chan<- PrometheusMetric) (float64, error) { return s.GetDiskSemp1(ch) },
	"GetEnvironmentSemp1":      func(s *Semp, ch chan<- PrometheusMetric) (float64, error) { return s.GetEnvironmentSemp1(ch) },
	"GetGlobalStatsSemp1":      func(s *Semp, ch chan<- PrometheusMetric) (float64, error) { return s.GetGlobalStatsSemp1(ch) },
	"GetGlobalSystemInfoSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) { return s.GetGlobalSystemInfoSemp1(ch) },
	"GetHardwareSemp1":         func(s *Semp, ch chan<- PrometheusMetric) (float64, error) { return s.GetHardwareSemp1(ch) },
	"GetHealthSemp1":           func(s *Semp, ch chan<- PrometheusMetric) (float64, error) { return s.GetHealthSemp1(ch) },
	"GetInterfaceHWSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetInterfaceHWSemp1(ch, goldenItemFilter)
	},
	"GetInterfaceSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetInterfaceSemp1(ch, goldenItemFilter)
	},
	"GetMemorySemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) { return s.GetMemorySemp1(ch) },
	"GetMqttSessionSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetMqttSessionSemp1(ch, goldenVpnFilter, goldenItemFilter, goldenPageSize)
	},
	"GetQueueDetailsSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetQueueDetailsSemp1(ch, goldenVpnFilter, goldenItemFilter, goldenPageSize)
	},
	"GetQueueRatesSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetQueueRatesSemp1(ch, goldenVpnFilter, goldenItemFilter, goldenPageSize)
	},
	"GetQueueStatsSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetQueueStatsSemp1(ch, goldenVpnFilter, goldenItemFilter, goldenPageSize)
	},
	"GetRaidSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) { return s.GetRaidSemp1(ch) },
	"GetRdpInfoSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetRdpInfoSemp1(ch, goldenVpnFilter, goldenItemFilter)
	},
	"GetRdpStatsSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetRdpStatsSemp1(ch, goldenVpnFilter, goldenItemFilter, goldenPageSize)
	},
	"GetRedundancySemp1":       func(s *Semp, ch chan<- PrometheusMetric) (float64, error) { return s.GetRedundancySemp1(ch) },
	"GetReplicationStatsSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) { return s.GetReplicationStatsSemp1(ch) },
	"GetRestConsumerStatsSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetRestConsumerStatsSemp1(ch, goldenVpnFilter, goldenItemFilter, goldenPageSize)
	},
	"GetSpoolSemp1":      func(s *Semp, ch chan<- PrometheusMetric) (float64, error) { return s.GetSpoolSemp1(ch) },
	"GetSpoolStatsSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) { return s.GetSpoolStatsSemp1(ch) },
	"GetStatsClientDetailSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetStatsClientDetailSemp1(ch)
	},
	"GetStorageElementSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetStorageElementSemp1(ch, goldenItemFilter)
	},
	"GetTopicEndpointDetailsSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetTopicEndpointDetailsSemp1(ch, goldenVpnFilter, goldenItemFilter, goldenPageSize)
	},
	"GetTopicEndpointRatesSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetTopicEndpointRatesSemp1(ch, goldenVpnFilter, goldenItemFilter, goldenPageSize)
	},
	"GetTopicEndpointStatsSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetTopicEndpointStatsSemp1(ch, goldenVpnFilter, goldenItemFilter, goldenPageSize)
	},
	"GetVersionSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) { return s.GetVersionSemp1(ch) },
	"GetVpnReplicationSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetVpnReplicationSemp1(ch, goldenVpnFilter)
	},
	"GetVpnSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetVpnSemp1(ch, goldenVpnFilter, goldenPageSize)
	},
	"GetVpnSpoolSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetVpnSpoolSemp1(ch, goldenVpnFilter, goldenPageSize)
	},
	"GetVpnStatsSemp1": func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
		return s.GetVpnStatsSemp1(ch, goldenVpnFilter, goldenPageSize)
	},
}

// fakeBroker answers every request through respond and records method, URL and body of each request.
type fakeBroker struct {
	mu       sync.Mutex
	requests []string
}

func newFakeBroker(t *testing.T, isHWBroker bool, respond func(path, body string) (int, string)) (*Semp, *fakeBroker) {
	t.Helper()
	fb := &fakeBroker{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		fb.mu.Lock()
		fb.requests = append(fb.requests, r.Method+" "+r.URL.RequestURI()+" "+string(raw))
		fb.mu.Unlock()
		status, reply := respond(r.URL.RequestURI(), string(raw))
		// Replies may reference the broker itself (SEMP v2 nextPageUri); its port is only known at runtime.
		reply = strings.ReplaceAll(reply, "{{BROKER}}", "http://"+r.Host)
		w.WriteHeader(status)
		_, _ = io.WriteString(w, reply)
	}))
	t.Cleanup(server.Close)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewSemp(logger, server.URL, http.Client{}, nil, false, isHWBroker), fb
}

// runScenario executes call against a fake broker and renders the observable outcome as text.
func runScenario(t *testing.T, title string, call getterCall, respond func(path, body string) (int, string)) string {
	t.Helper()
	return runScenarioOn(t, title, false, call, respond)
}

func runScenarioOn(t *testing.T, title string, isHWBroker bool, call getterCall, respond func(path, body string) (int, string)) string {
	t.Helper()
	s, fb := newFakeBroker(t, isHWBroker, respond)
	ch := make(chan PrometheusMetric, 100000)
	up, err := call(s, ch)
	close(ch)

	var sb strings.Builder
	fmt.Fprintf(&sb, "## %s\n", title)
	for i, req := range fb.requests {
		fmt.Fprintf(&sb, "request[%d]: %s\n", i+1, req)
	}
	fmt.Fprintf(&sb, "up: %s\n", strconv.FormatFloat(up, 'g', -1, 64))
	fmt.Fprintf(&sb, "err: %v\n", err)
	sb.WriteString(renderMetrics(ch))
	return sb.String()
}

// renderMetrics serializes metrics sorted by name. The sort is stable, so the emission order of metrics sharing a
// name (relevant for Collect's last-value-wins de-duplication) is preserved.
func renderMetrics(ch <-chan PrometheusMetric) string {
	var metrics []PrometheusMetric
	for m := range ch {
		metrics = append(metrics, m)
	}
	sort.SliceStable(metrics, func(i, j int) bool { return metrics[i].Name() < metrics[j].Name() })

	var sb strings.Builder
	for _, m := range metrics {
		valueType := "gauge"
		if m.valueType == prometheus.CounterValue {
			valueType = "counter"
		}
		fmt.Fprintf(&sb, "metric: %s %s %s\n", m.Name(), valueType, strconv.FormatFloat(m.value, 'g', -1, 64))
	}
	return sb.String()
}

func compareGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", "golden", name+".golden")
	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading golden file (run with -update to create it): %v", err)
	}
	if string(want) != got {
		t.Errorf("output differs from %s (run with -update after an intended change)\n--- got ---\n%s", path, got)
	}
}

func TestGoldenSemp1Getters(t *testing.T) {
	t.Parallel()

	fixtures, err := filepath.Glob(filepath.Join("testdata", "semp1", "*.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures) != len(semp1Getters) {
		t.Fatalf("%d fixtures but %d getters registered: every fixture needs a getter call and vice versa", len(fixtures), len(semp1Getters))
	}

	for name, call := range semp1Getters {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			raw, err := os.ReadFile(filepath.Join("testdata", "semp1", name+".xml"))
			if err != nil {
				t.Fatal(err)
			}
			fixture := string(raw)
			singlePage := strings.Replace(fixture, "{{MORE_COOKIE}}", "", 1)
			firstPage := strings.Replace(fixture, "{{MORE_COOKIE}}", "<more-cookie>"+page2Marker+"</more-cookie>", 1)

			var out strings.Builder
			out.WriteString(runScenario(t, "ok", call, func(_, _ string) (int, string) {
				return http.StatusOK, singlePage
			}))
			out.WriteString(runScenario(t, "paged", call, func(_, body string) (int, string) {
				if body == page2Marker {
					return http.StatusOK, singlePage
				}
				return http.StatusOK, firstPage
			}))
			out.WriteString(runScenario(t, "http-500", call, func(_, _ string) (int, string) {
				return http.StatusInternalServerError, "boom"
			}))
			out.WriteString(runScenario(t, "malformed-xml", call, func(_, _ string) (int, string) {
				return http.StatusOK, "<rpc-reply><rpc>"
			}))
			out.WriteString(runScenario(t, "execute-result-fail", call, func(_, _ string) (int, string) {
				return http.StatusOK, `<rpc-reply semp-version="soltr/10_0VMR"><execute-result code="fail" reason="golden failure"/></rpc-reply>`
			}))
			compareGolden(t, name, out.String())
		})
	}
}

// TestGoldenSemp1Scenarios replays the hand-written, realistic replies in testdata/semp1-scenarios. They cover the
// branches that depend on concrete string values (sensor types, card types, roles, certificate text) which the
// synthetic fixtures cannot reach. File names are <Getter>-<scenario>.xml; each runs against a software and a
// hardware broker.
func TestGoldenSemp1Scenarios(t *testing.T) {
	t.Parallel()

	files, err := filepath.Glob(filepath.Join("testdata", "semp1-scenarios", "*.xml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		base := strings.TrimSuffix(filepath.Base(file), ".xml")
		getter, _, _ := strings.Cut(base, "-")
		call, ok := semp1Getters[getter]
		if !ok {
			t.Fatalf("%s: no getter %q registered", file, getter)
		}
		t.Run(base, func(t *testing.T) {
			t.Parallel()
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			respond := func(_, _ string) (int, string) { return http.StatusOK, string(raw) }
			var out strings.Builder
			out.WriteString(runScenarioOn(t, "software-broker", false, call, respond))
			out.WriteString(runScenarioOn(t, "hardware-broker", true, call, respond))
			compareGolden(t, filepath.Join("scenarios", base), out.String())
		})
	}
}

// semp2QueueReply builds a SEMP v2 queue monitor reply; nextPageURI is empty for the last page.
func semp2QueueReply(nextPageURI string) string {
	queue := func(name string, base int) string {
		fields := []string{"spooledByteCount", "spooledMsgCount", "redeliveredMsgCount", "transportRetransmitMsgCount",
			"maxMsgSpoolUsageExceededDiscardedMsgCount", "maxMsgSizeExceededDiscardedMsgCount", "disabledDiscardedMsgCount",
			"destinationGroupErrorDiscardedMsgCount", "lowPriorityMsgCongestionDiscardedMsgCount", "deletedMsgCount",
			"maxTtlExpiredDiscardedMsgCount", "maxTtlExpiredToDmqMsgCount", "maxTtlExpiredToDmqFailedMsgCount",
			"maxRedeliveryExceededDiscardedMsgCount", "maxRedeliveryExceededToDmqMsgCount",
			"maxRedeliveryExceededToDmqFailedMsgCount", "txUnackedMsgCount", "xaTransactionNotSupportedDiscardedMsgCount"}
		parts := []string{`"queueName":"` + name + `"`, `"msgVpnName":"vpnA"`}
		for i, f := range fields {
			parts = append(parts, fmt.Sprintf("%q:%d", f, base+i))
		}
		return "{" + strings.Join(parts, ",") + "}"
	}
	return `{"data":[` + queue("q1", 100) + "," + queue("q2", 200) + "," + queue("q2", 200) + `],` +
		`"meta":{"count":2,"responseCode":200,"paging":{"nextPageUri":"` + nextPageURI + `"}}}`
}

func TestGoldenSemp2QueueStats(t *testing.T) {
	t.Parallel()

	ok := func(_, _ string) (int, string) { return http.StatusOK, semp2QueueReply("") }
	call := func(vpn, item string, metricFilter []string) getterCall {
		return func(s *Semp, ch chan<- PrometheusMetric) (float64, error) {
			return s.GetQueueStatsSemp2(ch, vpn, item, metricFilter)
		}
	}

	var out strings.Builder
	out.WriteString(runScenario(t, "ok-wildcard-item", call("vpnA", "*", nil), ok))
	out.WriteString(runScenario(t, "ok-item-name", call("vpnA", "my queue", nil), ok))
	out.WriteString(runScenario(t, "ok-item-where-expression", call("vpnA", "queueName==a*", nil), ok))
	out.WriteString(runScenario(t, "ok-metric-filter", call("vpnA", "*", []string{"solace_queue_msg_spool_usage_exceeded", "deletedMsgCount"}), ok))
	out.WriteString(runScenario(t, "invalid-metric-filter", call("vpnA", "*", []string{"no_such_metric"}), ok))
	out.WriteString(runScenario(t, "paged", call("vpnA", "*", nil), func(path, _ string) (int, string) {
		if strings.Contains(path, "cursor=page2") {
			return http.StatusOK, semp2QueueReply("")
		}
		return http.StatusOK, semp2QueueReply("{{BROKER}}/SEMP/v2/monitor/msgVpns/vpnA/queues?count=100&cursor=page2")
	}))
	out.WriteString(runScenario(t, "http-400-with-semp-error", call("vpnA", "*", nil), func(_, _ string) (int, string) {
		return http.StatusBadRequest, `{"meta":{"responseCode":400,"error":{"code":11,"description":"golden failure","status":"NOT_FOUND"}}}`
	}))
	out.WriteString(runScenario(t, "http-500", call("vpnA", "*", nil), func(_, _ string) (int, string) {
		return http.StatusInternalServerError, "boom"
	}))
	out.WriteString(runScenario(t, "malformed-json", call("vpnA", "*", nil), func(_, _ string) (int, string) {
		return http.StatusOK, "{"
	}))
	compareGolden(t, "GetQueueStatsSemp2", out.String())
}

// TestGoldenMetricDescriptions pins every registered metric description (registry key, name, help text, labels and
// SEMP v2 field), so dashboards and alerts relying on them cannot break unnoticed.
func TestGoldenMetricDescriptions(t *testing.T) {
	t.Parallel()

	var lines []string
	for group, descriptions := range MetricDesc {
		for key, d := range descriptions {
			lines = append(lines, fmt.Sprintf("%s/%s: %s labels=%v sempV2=%q help=%q",
				group, key, d.fqName, d.variableLabels, d.sempV2field, d.help))
		}
	}
	sort.Strings(lines)
	compareGolden(t, "metric_descriptions", strings.Join(lines, "\n")+"\n")
}
