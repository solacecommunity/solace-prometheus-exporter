package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"solace_exporter/internal/exporter"
	"strings"
	"sync"
	"testing"
	"time"
)

// queueReplyXML is a minimal valid SEMP1 getQueueDetails reply, enough for a scrape to succeed on correct auth.
const queueReplyXML = `<rpc-reply semp-version="soltr/9_1_1VMR"><rpc><show><queue><queues><queue><name>q1</name>` +
	`<info><message-vpn>default</message-vpn></info></queue></queues></queue></show></rpc>` +
	`<execute-result code="ok"/></rpc-reply>`

// mockBroker is a fake Solace SEMP endpoint that records the Basic-auth credentials it receives and rejects any
// request whose credentials are not its own (as a real broker's LDAP does with a 401).
type mockBroker struct {
	server   *httptest.Server
	user     string
	pass     string
	mu       sync.Mutex
	seen     map[string]int
	requests int
}

func newMockBroker(t *testing.T, idx int) *mockBroker {
	t.Helper()
	b := &mockBroker{
		user: fmt.Sprintf("user-%d", idx),
		pass: fmt.Sprintf("pass-%d", idx),
		seen: map[string]int{},
	}
	b.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, _ := r.BasicAuth()

		b.mu.Lock()
		b.seen[u+":"+p]++
		b.requests++
		b.mu.Unlock()

		if u != b.user || p != b.pass {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("Unauthorized"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(queueReplyXML))
	}))
	t.Cleanup(b.server.Close)
	return b
}

// TestConcurrentScrapesDoNotLeakCredentials is the regression test for the credential race: many concurrent /solace
// scrapes, each carrying its own broker credentials + scrapeURI, must never authenticate to a broker with another broker's
// credentials. On the buggy code (a single shared *Config mutated per request) this fails both under `-race`
// (concurrent write/read of conf.Username/Password/ScrapeURI) and functionally (brokers observe foreign creds).
func TestConcurrentScrapesDoNotLeakCredentials(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	const numBrokers = 8
	brokers := make([]*mockBroker, numBrokers)
	for i := range brokers {
		brokers[i] = newMockBroker(t, i)
	}

	// Base credentials that are WRONG for every broker: if a request ever fell back to (or was clobbered with) the
	// shared base Config, the receiving broker would record these and the test would fail.
	base := &exporter.Config{
		Username:   "base-should-never-be-used",
		Password:   "base-should-never-be-used",
		Timeout:    5 * time.Second,
		DefaultVpn: "default",
	}

	dataSource := []exporter.DataSource{{Name: "QueueDetails", VpnFilter: "*", ItemFilter: "*"}}

	const requestsPerBroker = 60
	var wg sync.WaitGroup
	for round := 0; round < requestsPerBroker; round++ {
		for i := range brokers {
			wg.Add(1)
			go func(b *mockBroker) {
				defer wg.Done()
				form := url.Values{}
				form.Set("username", b.user)
				form.Set("password", b.pass)
				form.Set("scrapeURI", b.server.URL)

				req := httptest.NewRequest(http.MethodPost, "/solace", strings.NewReader(form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				doHandle(httptest.NewRecorder(), req, dataSource, base, logger)
			}(brokers[i])
		}
	}
	wg.Wait()

	for i, b := range brokers {
		b.mu.Lock()
		if b.requests == 0 {
			t.Errorf("broker %d (%s) received no requests", i, b.user)
		}
		want := b.user + ":" + b.pass
		for creds, count := range b.seen {
			if creds != want {
				t.Errorf("CREDENTIAL LEAK: broker %d (%s) received foreign credentials %q %d time(s)", i, b.user, creds, count)
			}
		}
		b.mu.Unlock()
	}

	// The shared base Config must be pristine after all concurrent requests.
	if base.Username != "base-should-never-be-used" || base.Password != "base-should-never-be-used" || base.ScrapeURI != "" {
		t.Errorf("shared base Config was mutated by request handling: %+v", struct{ U, P, S string }{base.Username, base.Password, base.ScrapeURI})
	}
}
