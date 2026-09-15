package semp

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// statsClientDetailFields lists every element the getter reads, paired with a value that is unique across the whole
// set. Uniqueness is the point: with 32 near-identical `ch <- semp.NewMetric(...)` lines, a copy-paste slip that
// pairs the egress description with the ingress field would still emit a plausible metric, and only distinct
// per-field values expose it.
var statsClientDetailFields = []struct {
	element string
	value   float64
}{
	// Msg-Rate current (per-second)
	{"current-ingress-rate-per-second", 1},
	{"current-egress-rate-per-second", 2},
	{"current-ingress-persistent-rate-per-second", 3},
	{"current-egress-persistent-rate-per-second", 4},
	{"current-ingress-nonpersistent-rate-per-second", 5},
	{"current-egress-nonpersistent-rate-per-second", 6},
	{"current-ingress-direct-rate-per-second", 7},
	{"current-egress-direct-rate-per-second", 8},

	// Msg-Rate average (per-minute)
	{"average-ingress-rate-per-minute", 9},
	{"average-egress-rate-per-minute", 10},
	{"average-ingress-persistent-rate-per-minute", 11},
	{"average-egress-persistent-rate-per-minute", 12},
	{"average-ingress-nonpersistent-rate-per-minute", 13},
	{"average-egress-nonpersistent-rate-per-minute", 14},
	{"average-ingress-direct-rate-per-minute", 15},
	{"average-egress-direct-rate-per-minute", 16},

	// Byte-Rate current (per-second)
	{"current-ingress-byte-rate-per-second", 17},
	{"current-egress-byte-rate-per-second", 18},
	{"current-ingress-persistent-byte-rate-per-second", 19},
	{"current-egress-persistent-byte-rate-per-second", 20},
	{"current-ingress-nonpersistent-byte-rate-per-second", 21},
	{"current-egress-nonpersistent-byte-rate-per-second", 22},
	{"current-ingress-direct-byte-rate-per-second", 23},
	{"current-egress-direct-byte-rate-per-second", 24},

	// Byte-Rate average (per-minute)
	{"average-ingress-byte-rate-per-minute", 25},
	{"average-egress-byte-rate-per-minute", 26},
	{"average-ingress-persistent-byte-rate-per-minute", 27},
	{"average-egress-persistent-byte-rate-per-minute", 28},
	{"average-ingress-nonpersistent-byte-rate-per-minute", 29},
	{"average-egress-nonpersistent-byte-rate-per-minute", 30},
	{"average-ingress-direct-byte-rate-per-minute", 31},
	{"average-egress-direct-byte-rate-per-minute", 32},
}

// statsClientDetailReply wraps a stats body in the envelope `show stats client detail` returns.
func statsClientDetailReply(statsBody string, executeResult string) string {
	return `<rpc-reply semp-version="soltr/9_1_1VMR"><rpc><show><stats><client><global><stats>` +
		statsBody +
		`</stats></global></client></stats></show></rpc>` + executeResult + `</rpc-reply>`
}

func allStatsClientDetailFields() string {
	elements := make([]string, 0, len(statsClientDetailFields))
	for _, f := range statsClientDetailFields {
		elements = append(elements, "<"+f.element+">"+strconv.Itoa(int(f.value))+"</"+f.element+">")
	}
	return strings.Join(elements, "")
}

func newStatsClientDetailTestSemp(t *testing.T, reply string) *Semp {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(server.Close)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	return NewSemp(logger, server.URL, http.Client{}, nil, false, false)
}

// TestGetStatsClientDetailSemp1MapsEveryFieldToItsOwnMetric pins each SEMP element to the metric it must land on.
// It exists alongside the captured-payload test because the real capture cannot do this job: 22 of its 32 fields
// share a value with another field (12 are zero, and totals equal their direct counterparts on a direct-only
// broker), so transposing a pair there is undetectable. Synthetic values 1..32 are distinct by construction.
func TestGetStatsClientDetailSemp1MapsEveryFieldToItsOwnMetric(t *testing.T) {
	t.Parallel()
	s := newStatsClientDetailTestSemp(t, statsClientDetailReply(allStatsClientDetailFields(), `<execute-result code="ok"/>`))

	ch := make(chan PrometheusMetric, 100)
	up, err := s.GetStatsClientDetailSemp1(ch)
	metrics := drain(ch)

	if err != nil {
		t.Fatalf("GetStatsClientDetailSemp1 error: %v", err)
	}
	if up != 1 {
		t.Errorf("up = %v, want 1", up)
	}
	if len(metrics) != len(statsClientDetailFields) {
		t.Errorf("emitted %d metrics, want %d", len(metrics), len(statsClientDetailFields))
	}

	got := make(map[string]float64, len(metrics))
	for _, m := range metrics {
		if _, dupe := got[m.Name()]; dupe {
			t.Errorf("metric %s emitted more than once", m.Name())
		}
		got[m.Name()] = m.value
	}

	// Registry keys are the element name with dashes turned to underscores; the metric name comes from the registry,
	// since it deliberately differs from the element name (plural base units, no redundant "rate").
	for _, f := range statsClientDetailFields {
		key := strings.ReplaceAll(f.element, "-", "_")
		desc, registered := MetricDesc["StatsClientDetail"][key]
		if !registered {
			t.Errorf("element <%s> has no registry entry under key %q", f.element, key)
			continue
		}
		value, ok := got[desc.fqName]
		if !ok {
			t.Errorf("missing metric %s (element <%s>)", desc.fqName, f.element)
			continue
		}
		if value != f.value {
			t.Errorf("%s = %v, want %v (value came from the wrong SEMP element)", desc.fqName, value, f.value)
		}
	}
}

// TestGetStatsClientDetailSemp1RejectsNotOkResult covers the recoverable-failure branch: a broker that refuses the
// command (for example one that does not support `detail`) must report up=0 with an error rather than publishing a
// full set of zeroes.
func TestGetStatsClientDetailSemp1RejectsNotOkResult(t *testing.T) {
	t.Parallel()
	s := newStatsClientDetailTestSemp(t, statsClientDetailReply("", `<execute-result code="fail" reason="unknown command"/>`))

	ch := make(chan PrometheusMetric, 100)
	up, err := s.GetStatsClientDetailSemp1(ch)
	metrics := drain(ch)

	if err == nil {
		t.Error("expected an error for a non-ok execute-result")
	}
	if up != 0 {
		t.Errorf("up = %v, want 0 (recoverable failure for this target only)", up)
	}
	if len(metrics) != 0 {
		t.Errorf("emitted %d metrics on a failed command, want 0", len(metrics))
	}
}

// capturedPayload is a real `show stats client detail` reply from a SolOS 10.25 appliance under production load.
const capturedPayload = "../../test/data/semp1-stats-client-detail.txt"

// leafValuesUnderStatsBlock extracts every numeric leaf element directly under rpc/show/stats/client/global/stats,
// walking the XML generically rather than through the getter's own Data struct. That independence is the whole point:
// decoding the fixture with the same struct tags the getter uses would make a mistyped tag agree with itself and pass.
// Here the element names come from the broker, the metric names from MetricDesc, and the tags are what gets tested.
func leafValuesUnderStatsBlock(t *testing.T, raw []byte) map[string]float64 {
	t.Helper()

	statsPath := []string{"rpc-reply", "rpc", "show", "stats", "client", "global", "stats"}
	values := make(map[string]float64)
	var path []string

	decoder := xml.NewDecoder(bytes.NewReader(raw))
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("walking %s: %v", capturedPayload, err)
		}

		switch element := token.(type) {
		case xml.StartElement:
			path = append(path, element.Name.Local)
			if len(path) != len(statsPath)+1 || !slices.Equal(path[:len(statsPath)], statsPath) {
				continue
			}

			// Probe the element: containers such as <zip-stats> are skipped, leaves are recorded.
			var probe struct {
				Text     string `xml:",chardata"`
				InnerXML string `xml:",innerxml"`
			}
			if err := decoder.DecodeElement(&probe, &element); err != nil {
				t.Fatalf("decoding <%s>: %v", element.Name.Local, err)
			}
			path = path[:len(path)-1] // DecodeElement consumed the matching end element
			if strings.Contains(probe.InnerXML, "<") {
				continue
			}
			if value, err := strconv.ParseFloat(strings.TrimSpace(probe.Text), 64); err == nil {
				values[element.Name.Local] = value
			}
		case xml.EndElement:
			path = path[:len(path)-1]
		}
	}

	return values
}

// TestGetStatsClientDetailSemp1AgainstCapturedBrokerPayload checks the getter's xml tags against what a real broker
// actually sends. A tag that does not match an element the broker emits is invisible to the synthetic test above,
// because that test builds its fixture from those same tags; here a mismatch shows up as a zero where the captured
// payload has a real number.
func TestGetStatsClientDetailSemp1AgainstCapturedBrokerPayload(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(capturedPayload)
	if err != nil {
		t.Fatalf("reading captured payload: %v", err)
	}
	broker := leafValuesUnderStatsBlock(t, raw)

	s := newStatsClientDetailTestSemp(t, string(raw))
	ch := make(chan PrometheusMetric, 200)
	up, err := s.GetStatsClientDetailSemp1(ch)
	metrics := drain(ch)

	if err != nil {
		t.Fatalf("GetStatsClientDetailSemp1 error: %v", err)
	}
	if up != 1 {
		t.Errorf("up = %v, want 1", up)
	}

	got := make(map[string]float64, len(metrics))
	for _, m := range metrics {
		got[m.Name()] = m.value
	}

	// Registry keys are the SEMP element name with dashes swapped for underscores; assert that holds against the
	// real payload, so a key naming an element no broker sends is caught rather than silently publishing zero.
	verified := 0
	for key, desc := range MetricDesc["StatsClientDetail"] {
		element := strings.ReplaceAll(key, "_", "-")
		brokerValue, present := broker[element]
		if !present {
			t.Errorf("registry key %q implies element <%s>, absent from the captured payload", key, element)
			continue
		}
		gotValue, emitted := got[desc.fqName]
		if !emitted {
			t.Errorf("no metric emitted for %s", desc.fqName)
			continue
		}
		if gotValue != brokerValue {
			t.Errorf("%s = %v, want %v from <%s>", desc.fqName, gotValue, brokerValue, element)
		}
		if brokerValue != 0 {
			verified++
		}
	}

	// Only non-zero fields genuinely prove their tag: a wrong tag on a field the broker reports as 0 also decodes to
	// 0, so the assertion above cannot tell them apart. The synthetic test covers the wiring of those fields.
	t.Logf("%d of %d fields carry a non-zero value in this capture and are fully verified by it",
		verified, len(MetricDesc["StatsClientDetail"]))
}
