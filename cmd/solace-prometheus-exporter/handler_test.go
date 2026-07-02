package main

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"solace_exporter/internal/exporter"
	"strings"
	"testing"
	"time"
)

var solaceUpRe = regexp.MustCompile(`(?m)^solace_up\{[^}]*\}\s+([0-9.]+)`)

func scrapeUp(t *testing.T, body string) string {
	t.Helper()
	m := solaceUpRe.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no solace_up metric found in response body:\n%s", body)
	}
	return m[1]
}

// TestDoHandleEndToEnd exercises the full /solace scrape path (request -> per-request config -> exporter -> SEMP)
// and asserts solace_up reflects authentication success/failure with the per-request credentials.
func TestDoHandleEndToEnd(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	broker := newMockBroker(t, 42) // expects user-42 / pass-42
	base := &exporter.Config{Username: "wrong-base", Password: "wrong-base", Timeout: 5 * time.Second, DefaultVpn: "default"}
	ds := []exporter.DataSource{{Name: "QueueDetails", VpnFilter: "*", ItemFilter: "*"}}

	do := func(user, pass string) string {
		form := url.Values{}
		form.Set("username", user)
		form.Set("password", pass)
		form.Set("scrapeURI", broker.server.URL)
		req := httptest.NewRequest(http.MethodPost, "/solace", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()
		doHandle(rr, req, ds, base, logger)
		return rr.Body.String()
	}

	t.Run("correct credentials -> up 1", func(t *testing.T) {
		if up := scrapeUp(t, do("user-42", "pass-42")); up != "1" {
			t.Errorf("solace_up = %s, want 1 (broker should accept correct credentials)", up)
		}
	})

	t.Run("wrong credentials -> up 0", func(t *testing.T) {
		if up := scrapeUp(t, do("user-42", "bad-pass")); up != "0" {
			t.Errorf("solace_up = %s, want 0 (broker should 401 wrong credentials)", up)
		}
	})
}

// TestDoHandleExporterAuthProtectsEndpoint verifies the exporter endpoint itself can be protected with basic auth
// (SOLACE_EXPORTER_AUTH_*), independent of the broker credentials.
func TestDoHandleExporterAuthProtectsEndpoint(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	broker := newMockBroker(t, 7)

	base := &exporter.Config{
		Username:     "user-7",
		Password:     "pass-7",
		ScrapeURI:    broker.server.URL,
		Timeout:      5 * time.Second,
		DefaultVpn:   "default",
		ExporterAuth: exporter.ExporterAuthConfig{Scheme: "basic", Username: "scrape-user", Password: "scrape-pass"},
	}
	ds := []exporter.DataSource{{Name: "QueueDetails", VpnFilter: "*", ItemFilter: "*"}}

	newReq := func() *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/solace", nil)
		return req
	}

	// Without exporter credentials -> 401.
	rr := httptest.NewRecorder()
	doHandle(rr, newReq(), ds, base, logger)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("without exporter auth: status = %d, want 401", rr.Code)
	}

	// With correct exporter credentials -> allowed (200) and a scrape happens.
	rr = httptest.NewRecorder()
	req := newReq()
	req.SetBasicAuth("scrape-user", "scrape-pass")
	doHandle(rr, req, ds, base, logger)
	if rr.Code != http.StatusOK {
		t.Errorf("with exporter auth: status = %d, want 200", rr.Code)
	}
}
