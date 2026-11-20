package exporter

// Characterization test for the scrape-target dispatch in CollectPrometheusMetric: for every target name (and its
// V1 alias) on a software and a hardware broker it records which SEMP request is sent and which metrics, including
// solace_up, are emitted. Regenerate after an intended change with:
//
//	go test ./internal/exporter -run TestGoldenDispatch -update

import (
	"context"
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
	"time"

	"solace_exporter/internal/semp"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

var updateGolden = flag.Bool("update", false, "rewrite the golden files in testdata/golden")

// dispatchTargets lists every scrape-target name CollectPrometheusMetric understands, plus an unknown one.
var dispatchTargets = []string{
	"Version", "VersionV1", "Health", "HealthV1", "StorageElement", "StorageElementV1", "Disk", "DiskV1",
	"Raid", "RaidV1", "Memory", "MemoryV1", "Interface", "InterfaceV1", "InterfaceHW", "InterfaceHWV1",
	"GlobalStats", "GlobalStatsV1", "GlobalSystemInfo", "GlobalSystemInfoV1", "Spool", "SpoolV1",
	"SpoolStats", "SpoolStatsV1", "Redundancy", "RedundancyV1", "Alarm", "AlarmV1", "Environment", "EnvironmentV1",
	"Hardware", "HardwareV1", "ClockDetail", "ClockDetailV1", "ReplicationStats", "ReplicationStatsV1",
	"ConfigSyncRouter", "ConfigSyncRouterV1", "ConfigSync", "ConfigSyncV1", "Vpn", "VpnV1",
	"VpnReplication", "VpnReplicationV1", "ConfigSyncVpn", "ConfigSyncVpnV1", "Bridge", "BridgeV1",
	"BridgeRemote", "BridgeRemoteV1", "BridgeDetail", "BridgeDetailV1", "BridgeClientCert", "BridgeClientCertV1",
	"VpnSpool", "VpnSpoolV1", "Client", "ClientV1", "ClientProfile", "ClientProfileV1",
	"ClientSlowSubscriber", "ClientSlowSubscriberV1", "ClientStats", "ClientStatsV1",
	"ClientConnections", "ClientConnectionsV1", "ClientMessageSpoolStats", "ClientMessageSpoolStatsV1",
	"ClientMessageSpoolEgress", "ClientMessageSpoolEgressV1", "ClusterLinks", "ClusterLinksV1",
	"VpnStats", "VpnStatsV1", "BridgeStats", "BridgeStatsV1", "QueueRates", "QueueRatesV1",
	"QueueStats", "QueueStatsV1", "QueueStatsV2", "QueueDetails", "QueueDetailsV1",
	"TopicEndpointRates", "TopicEndpointRatesV1", "TopicEndpointStats", "TopicEndpointStatsV1",
	"TopicEndpointDetails", "TopicEndpointDetailsV1", "RestConsumerStats", "RestConsumerStatsV1",
	"RdpStats", "RdpStatsV1", "RdpInfo", "RdpInfoV1", "MqttSession", "StatsClientDetail", "StatsClientDetailV1",
	"NoSuchTarget", "vpn", // unknown names; matching is case-sensitive
}

// dispatchBroker is a fake broker answering every SEMP v1 request with an empty successful reply and every SEMP v2
// request with an empty data list; it records the requests it receives.
type dispatchBroker struct {
	server   *httptest.Server
	mu       sync.Mutex
	requests []string
}

func newDispatchBroker(t *testing.T) *dispatchBroker {
	t.Helper()
	b := &dispatchBroker{}
	b.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		b.mu.Lock()
		b.requests = append(b.requests, strings.TrimSpace(r.Method+" "+r.URL.RequestURI()+" "+string(raw)))
		b.mu.Unlock()
		if r.Method == http.MethodGet {
			_, _ = io.WriteString(w, `{"data":[],"meta":{"count":0,"responseCode":200}}`)
			return
		}
		_, _ = io.WriteString(w, `<rpc-reply semp-version="soltr/10_0"><rpc><show/></rpc><execute-result code="ok"/></rpc-reply>`)
	}))
	t.Cleanup(b.server.Close)
	return b
}

func newDispatchExporter(t *testing.T, scrapeURI string, isHWBroker bool, defaultVpn string, dataSources []DataSource) *Exporter {
	t.Helper()
	conf := &Config{
		ScrapeURI:    scrapeURI,
		Username:     "user",
		Password:     "pass",
		Timeout:      5 * time.Second,
		IsHWBroker:   isHWBroker,
		SempPageSize: 11,
		DefaultVpn:   defaultVpn,
	}
	if err := conf.DetermineAuthType(); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewExporter(context.Background(), logger, conf, &dataSources)
}

// collectAll runs CollectPrometheusMetric and renders the emitted metrics with their values, sorted by name
// (stable, so the emission order of metrics sharing a name is preserved).
func collectAll(e *Exporter) string {
	ch := make(chan semp.PrometheusMetric, 10000)
	e.CollectPrometheusMetric(ch)
	close(ch)
	var metrics []semp.PrometheusMetric
	for m := range ch {
		metrics = append(metrics, m)
	}
	sort.SliceStable(metrics, func(i, j int) bool { return metrics[i].Name() < metrics[j].Name() })
	var sb strings.Builder
	for _, m := range metrics {
		value := testutil.ToFloat64(&mockCollector{m: m.AsPrometheusMetric()})
		fmt.Fprintf(&sb, "metric: %s %s\n", m.Name(), strconv.FormatFloat(value, 'g', -1, 64))
	}
	return sb.String()
}

func runDispatch(t *testing.T, title string, isHWBroker bool, defaultVpn string, dataSources []DataSource) string {
	t.Helper()
	b := newDispatchBroker(t)
	out := collectAll(newDispatchExporter(t, b.server.URL, isHWBroker, defaultVpn, dataSources))
	var sb strings.Builder
	fmt.Fprintf(&sb, "## %s\n", title)
	for i, r := range b.requests {
		fmt.Fprintf(&sb, "request[%d]: %s\n", i+1, r)
	}
	sb.WriteString(out)
	return sb.String()
}

func TestGoldenDispatch(t *testing.T) {
	t.Parallel()

	var sb strings.Builder
	for _, name := range dispatchTargets {
		// Distinct filter values reveal which filter each target passes to its getter.
		ds := []DataSource{{Name: name, VpnFilter: "vpnX", ItemFilter: "itemY"}}
		sb.WriteString(runDispatch(t, name+" software-broker", false, "", ds))
		sb.WriteString(runDispatch(t, name+" hardware-broker", true, "", ds))
	}

	// SEMP v2 needs a concrete VPN: "*" falls back to defaultVpn, and fails without one.
	sb.WriteString(runDispatch(t, "QueueStatsV2 wildcard vpn with defaultVpn", false, "defVpn",
		[]DataSource{{Name: "QueueStatsV2", VpnFilter: "*", ItemFilter: "q1", MetricFilter: []string{"spooledMsgCount"}}}))
	sb.WriteString(runDispatch(t, "QueueStatsV2 wildcard vpn without defaultVpn", false, " ",
		[]DataSource{{Name: "QueueStatsV2", VpnFilter: "*", ItemFilter: "q1"}}))
	// A failing QueueStatsV2 after a successful target must not inherit the previous up value.
	sb.WriteString(runDispatch(t, "QueueStatsV2 failure after success", false, "",
		[]DataSource{{Name: "Version", VpnFilter: "*", ItemFilter: "*"}, {Name: "QueueStatsV2", VpnFilter: "*", ItemFilter: "*"}}))
	// Several targets in one scrape are processed in order.
	sb.WriteString(runDispatch(t, "multiple targets", false, "",
		[]DataSource{{Name: "Version", VpnFilter: "*", ItemFilter: "*"}, {Name: "Disk", VpnFilter: "*", ItemFilter: "*"},
			{Name: "Vpn", VpnFilter: "a", ItemFilter: "*"}}))

	// An unreachable broker is unrecoverable: one global solace_up and the remaining targets are skipped.
	unreachable := httptest.NewServer(http.NotFoundHandler())
	unreachableURL := unreachable.URL
	unreachable.Close()
	e := newDispatchExporter(t, unreachableURL, false, "",
		[]DataSource{{Name: "Version", VpnFilter: "*", ItemFilter: "*"}, {Name: "Memory", VpnFilter: "*", ItemFilter: "*"}})
	out := collectAll(e)
	// The connection error text contains the random port; keep the golden file stable.
	out = strings.ReplaceAll(out, strings.TrimPrefix(unreachableURL, "http://"), "127.0.0.1:PORT")
	sb.WriteString("## unreachable broker aborts remaining targets\n" + out)

	compareGolden(t, "dispatch", sb.String())
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

// TestEveryScrapeTargetIsCharacterized keeps dispatchTargets in sync with the registry, so a newly registered target
// cannot escape the dispatch golden test.
func TestEveryScrapeTargetIsCharacterized(t *testing.T) {
	t.Parallel()
	listed := make(map[string]bool, len(dispatchTargets))
	for _, name := range dispatchTargets {
		listed[name] = true
	}
	for name := range scrapeTargets {
		if !listed[name] {
			t.Errorf("scrape target %q is registered but missing from dispatchTargets", name)
		}
	}
}
