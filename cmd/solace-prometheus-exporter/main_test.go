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

func TestDoHandleHeaderOverride(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	dataSource := []exporter.DataSource{{Name: "test"}}

	tests := []struct {
		name              string
		queryParams       map[string]string
		headers           map[string]string
		initialConf       *exporter.Config
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
			initialConf:       &exporter.Config{Username: "conf-user", Password: "conf-pass", ScrapeURI: "http://conf-uri", Timeout: 5 * time.Second},
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
			initialConf:       &exporter.Config{Username: "conf-user", Password: "conf-pass", ScrapeURI: "http://conf-uri", Timeout: 5 * time.Second},
			expectedUsername:  "form-user",
			expectedPassword:  "form-pass",
			expectedScrapeURI: "http://form-uri",
			expectedTimeout:   15 * time.Second,
		},
		{
			name:              "Fallback to config",
			headers:           map[string]string{},
			initialConf:       &exporter.Config{Username: "conf-user", Password: "conf-pass", ScrapeURI: "http://conf-uri", Timeout: 5 * time.Second},
			expectedUsername:  "conf-user",
			expectedPassword:  "conf-pass",
			expectedScrapeURI: "http://conf-uri",
			expectedTimeout:   5 * time.Second,
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
			rr := httptest.NewRecorder()

			// Create a new config to avoid copying the mutex
			conf := exporter.Config{
				Username:  tt.initialConf.Username,
				Password:  tt.initialConf.Password,
				ScrapeURI: tt.initialConf.ScrapeURI,
				Timeout:   tt.initialConf.Timeout,
				// copy other fields if necessary for doHandle
				ExporterAuth: tt.initialConf.ExporterAuth,
			}
			// We need to pass a pointer to conf because doHandle modifies it.
			doHandle(rr, req, dataSource, &conf, logger)

			if conf.Username != tt.expectedUsername {
				t.Errorf("Username: expected %s, got %s", tt.expectedUsername, conf.Username)
			}
			if conf.Password != tt.expectedPassword {
				t.Errorf("Password: expected %s, got %s", tt.expectedPassword, conf.Password)
			}
			if conf.ScrapeURI != tt.expectedScrapeURI {
				t.Errorf("ScrapeURI: expected %s, got %s", tt.expectedScrapeURI, conf.ScrapeURI)
			}
			if conf.Timeout != tt.expectedTimeout {
				t.Errorf("Timeout: expected %v, got %v", tt.expectedTimeout, conf.Timeout)
			}
		})
	}
}
