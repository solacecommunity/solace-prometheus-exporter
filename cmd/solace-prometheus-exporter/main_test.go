package main

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"solace_exporter/internal/exporter"
	"testing"
	"time"
)

// TestResolveRequestConfig verifies that per-request credentials/scrapeURI/timeout are resolved with the correct
// precedence (form value > x-solace-broker-* header > configured base value) AND that the shared base Config is
// never mutated - which is the fix for the broker-wide SEMP 401 storm.
func TestResolveRequestConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name              string
		queryParams       map[string]string
		headers           map[string]string
		base              exporter.Config
		expectedUsername  string
		expectedPassword  string
		expectedScrapeURI string
		expectedTimeout   time.Duration
	}{
		{
			name: "Header override",
			headers: map[string]string{
				"x-solace-broker-username":  "header-user",
				"x-solace-broker-password":  "header-pass",
				"x-solace-broker-scrapeuri": "http://header-uri",
				"x-solace-broker-timeout":   "10s",
			},
			base:              exporter.Config{Username: "conf-user", Password: "conf-pass", ScrapeURI: "http://conf-uri", Timeout: 5 * time.Second},
			expectedUsername:  "header-user",
			expectedPassword:  "header-pass",
			expectedScrapeURI: "http://header-uri",
			expectedTimeout:   10 * time.Second,
		},
		{
			name: "FormValue takes priority over header",
			queryParams: map[string]string{
				"username":  "form-user",
				"password":  "form-pass",
				"scrapeURI": "http://form-uri",
				"timeout":   "15s",
			},
			headers: map[string]string{
				"x-solace-broker-username":  "header-user",
				"x-solace-broker-password":  "header-pass",
				"x-solace-broker-scrapeuri": "http://header-uri",
				"x-solace-broker-timeout":   "10s",
			},
			base:              exporter.Config{Username: "conf-user", Password: "conf-pass", ScrapeURI: "http://conf-uri", Timeout: 5 * time.Second},
			expectedUsername:  "form-user",
			expectedPassword:  "form-pass",
			expectedScrapeURI: "http://form-uri",
			expectedTimeout:   15 * time.Second,
		},
		{
			name:              "Fallback to config",
			headers:           map[string]string{},
			base:              exporter.Config{Username: "conf-user", Password: "conf-pass", ScrapeURI: "http://conf-uri", Timeout: 5 * time.Second},
			expectedUsername:  "conf-user",
			expectedPassword:  "conf-pass",
			expectedScrapeURI: "http://conf-uri",
			expectedTimeout:   5 * time.Second,
		},
		{
			name:              "Invalid timeout keeps configured timeout",
			queryParams:       map[string]string{"timeout": "not-a-duration"},
			base:              exporter.Config{Username: "conf-user", Password: "conf-pass", ScrapeURI: "http://conf-uri", Timeout: 7 * time.Second},
			expectedUsername:  "conf-user",
			expectedPassword:  "conf-pass",
			expectedScrapeURI: "http://conf-uri",
			expectedTimeout:   7 * time.Second,
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			url := "/"
			if len(tt.queryParams) > 0 {
				url += "?"
				for k, v := range tt.queryParams {
					url += k + "=" + v + "&"
				}
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			base := tt.base // keep a copy of the original to detect mutation
			reqConf := resolveRequestConfig(req, &base, logger)

			if reqConf.Username != tt.expectedUsername {
				t.Errorf("Username: expected %s, got %s", tt.expectedUsername, reqConf.Username)
			}
			if reqConf.Password != tt.expectedPassword {
				t.Errorf("Password: expected %s, got %s", tt.expectedPassword, reqConf.Password)
			}
			if reqConf.ScrapeURI != tt.expectedScrapeURI {
				t.Errorf("ScrapeURI: expected %s, got %s", tt.expectedScrapeURI, reqConf.ScrapeURI)
			}
			if reqConf.Timeout != tt.expectedTimeout {
				t.Errorf("Timeout: expected %v, got %v", tt.expectedTimeout, reqConf.Timeout)
			}

			// The shared base Config must NOT be mutated by request resolution.
			if base.Username != tt.base.Username || base.Password != tt.base.Password ||
				base.ScrapeURI != tt.base.ScrapeURI || base.Timeout != tt.base.Timeout {
				t.Errorf("base Config was mutated: got %+v, want %+v",
					struct {
						U, P, S string
						T       time.Duration
					}{base.Username, base.Password, base.ScrapeURI, base.Timeout},
					struct {
						U, P, S string
						T       time.Duration
					}{tt.base.Username, tt.base.Password, tt.base.ScrapeURI, tt.base.Timeout})
			}
		})
	}
}

func TestFirstNonEmpty(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   string
	}{
		{"all empty", []string{"", ""}, ""},
		{"first wins", []string{"a", "b"}, "a"},
		{"skip empty", []string{"", "b", "c"}, "b"},
		{"none", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := firstNonEmpty(tt.values...); got != tt.want {
				t.Errorf("firstNonEmpty(%v) = %q, want %q", tt.values, got, tt.want)
			}
		})
	}
}
