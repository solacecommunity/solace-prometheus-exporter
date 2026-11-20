package semp

import (
	"encoding/xml"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// injectedFilter tries to close its XML element, add an element of its own and inject a fmt verb.
const injectedFilter = `q&1'"</name></vpn-name><evil/>%d`

// filteredSemp1Getters lists every SEMP v1 getter that puts scrape filters into its command.
var filteredSemp1Getters = map[string]func(s *Semp, ch chan<- PrometheusMetric, vpn, item string){
	"Bridge": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) {
		_, _ = s.GetBridgeSemp1(ch, vpn, item, 10)
	},
	"BridgeClientCert": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) {
		_, _ = s.GetBridgeClientCertSemp1(ch, vpn, item, 10)
	},
	"BridgeDetail": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) {
		_, _ = s.GetBridgeDetailSemp1(ch, vpn, item, 10)
	},
	"BridgeRemote": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) {
		_, _ = s.GetBridgeRemoteSemp1(ch, vpn, item)
	},
	"BridgeStats": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) {
		_, _ = s.GetBridgeStatsSemp1(ch, vpn, item, 10)
	},
	"Client": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) { _, _ = s.GetClientSemp1(ch, vpn, item) },
	"ClientConnectionStats": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) {
		_, _ = s.GetClientConnectionStatsSemp1(ch, item)
	},
	"ClientMessageSpoolEgress": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) {
		_, _ = s.GetClientMessageSpoolEgressSemp1(ch, item)
	},
	"ClientMessageSpoolStats": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) {
		_, _ = s.GetClientMessageSpoolStatsSemp1(ch, item, 10)
	},
	"ClientProfile": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) { _, _ = s.GetClientProfileSemp1(ch, vpn) },
	"ClientSlowSubscriber": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) {
		_, _ = s.GetClientSlowSubscriberSemp1(ch, vpn, item)
	},
	"ClientStats": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) {
		_, _ = s.GetClientStatsSemp1(ch, item, 10)
	},
	"ClusterLinks": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) {
		_, _ = s.GetClusterLinksSemp1(ch, vpn, item)
	},
	"ConfigSyncVpn": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) {
		_, _ = s.GetConfigSyncVpnSemp1(ch, vpn, 10)
	},
	"Interface":   func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) { _, _ = s.GetInterfaceSemp1(ch, item) },
	"InterfaceHW": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) { _, _ = s.GetInterfaceHWSemp1(ch, item) },
	"MqttSession": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) {
		_, _ = s.GetMqttSessionSemp1(ch, vpn, item, 10)
	},
	"QueueDetails": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) {
		_, _ = s.GetQueueDetailsSemp1(ch, vpn, item, 10)
	},
	"QueueRates": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) {
		_, _ = s.GetQueueRatesSemp1(ch, vpn, item, 10)
	},
	"QueueStats": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) {
		_, _ = s.GetQueueStatsSemp1(ch, vpn, item, 10)
	},
	"RdpInfo": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) { _, _ = s.GetRdpInfoSemp1(ch, vpn, item) },
	"RdpStats": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) {
		_, _ = s.GetRdpStatsSemp1(ch, vpn, item, 10)
	},
	"RestConsumerStats": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) {
		_, _ = s.GetRestConsumerStatsSemp1(ch, vpn, item, 10)
	},
	"StorageElement": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) { _, _ = s.GetStorageElementSemp1(ch, item) },
	"TopicEndpointDetails": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) {
		_, _ = s.GetTopicEndpointDetailsSemp1(ch, vpn, item, 10)
	},
	"TopicEndpointRates": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) {
		_, _ = s.GetTopicEndpointRatesSemp1(ch, vpn, item, 10)
	},
	"TopicEndpointStats": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) {
		_, _ = s.GetTopicEndpointStatsSemp1(ch, vpn, item, 10)
	},
	"Vpn":            func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) { _, _ = s.GetVpnSemp1(ch, vpn, 10) },
	"VpnReplication": func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) { _, _ = s.GetVpnReplicationSemp1(ch, vpn) },
	"VpnSpool":       func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) { _, _ = s.GetVpnSpoolSemp1(ch, vpn, 10) },
	"VpnStats":       func(s *Semp, ch chan<- PrometheusMetric, vpn, item string) { _, _ = s.GetVpnStatsSemp1(ch, vpn, 10) },
}

// firstSemp1Command runs call against a broker that fails every request and returns the first command sent.
func firstSemp1Command(t *testing.T, call func(s *Semp, ch chan<- PrometheusMetric)) string {
	t.Helper()
	var mu sync.Mutex
	var commands []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		commands = append(commands, string(body))
		mu.Unlock()
		_, _ = io.WriteString(w, `<rpc-reply><execute-result code="fail" reason="test"/></rpc-reply>`)
	}))
	t.Cleanup(server.Close)
	s := NewSemp(slog.New(slog.NewTextHandler(io.Discard, nil)), server.URL, http.Client{}, nil, false, false)

	call(s, make(chan PrometheusMetric, 100))

	mu.Lock()
	defer mu.Unlock()
	if len(commands) == 0 {
		t.Fatal("getter sent no request")
	}
	return commands[0]
}

// decodeSemp1Command parses a command, failing on malformed XML, and returns its element names and text content.
func decodeSemp1Command(t *testing.T, command string) (elements []string, text string) {
	t.Helper()
	decoder := xml.NewDecoder(strings.NewReader(command))
	var sb strings.Builder
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return elements, sb.String()
		}
		if err != nil {
			t.Fatalf("command is not well-formed XML (%v): %s", err, command)
		}
		switch tok := token.(type) {
		case xml.StartElement:
			elements = append(elements, tok.Name.Local)
		case xml.CharData:
			sb.Write(tok)
		}
	}
}

// TestSemp1CommandsEscapeFilters ensures filters from the scrape request reach the broker as plain text: they can
// neither close their XML element nor be interpreted as fmt verbs.
func TestSemp1CommandsEscapeFilters(t *testing.T) {
	t.Parallel()
	for name, getter := range filteredSemp1Getters {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			command := firstSemp1Command(t, func(s *Semp, ch chan<- PrometheusMetric) {
				getter(s, ch, injectedFilter, injectedFilter)
			})

			elements, text := decodeSemp1Command(t, command)

			for _, element := range elements {
				if element == "evil" {
					t.Errorf("filter injected an XML element: %s", command)
				}
			}
			if !strings.Contains(text, injectedFilter) {
				t.Errorf("command text %q does not contain the filter %q verbatim", text, injectedFilter)
			}
		})
	}
}

func TestSemp1EscapeKeepsWellFormedFiltersUnchanged(t *testing.T) {
	t.Parallel()
	for _, filter := range []string{"*", "my-vpn", "#config-sync", "queue/a/b", "q_1.2", ""} {
		if got := semp1Escape(filter); got != filter {
			t.Errorf("semp1Escape(%q) = %q, want it unchanged", filter, got)
		}
	}
	if got := semp1Escape(`a&b<c>"d'`); got != "a&amp;b&lt;c&gt;&#34;d&#39;" {
		t.Errorf("semp1Escape escaped special characters as %q", got)
	}
}
