package exporter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// clearSolaceEnv removes all SOLACE_* / PREFETCH_INTERVAL env vars so a test starts from a known state, restoring
// them via t.Cleanup so tests stay independent of the ambient environment. It uses os.LookupEnv so a variable that
// was unset is restored to unset (not to an empty string), avoiding state leaking across tests.
func clearSolaceEnv(t *testing.T) {
	t.Helper()
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(key, "SOLACE_") || key == "PREFETCH_INTERVAL" {
			k := key
			old, had := os.LookupEnv(k)
			_ = os.Unsetenv(k)
			t.Cleanup(func() {
				if had {
					_ = os.Setenv(k, old)
				} else {
					_ = os.Unsetenv(k)
				}
			})
		}
	}
}

func TestParseConfigBasicAuthFromEnv(t *testing.T) {
	clearSolaceEnv(t)
	t.Setenv("SOLACE_SCRAPE_URI", "http://broker:8080")
	t.Setenv("SOLACE_USERNAME", "monitor")
	t.Setenv("SOLACE_PASSWORD", "secret")

	endpoints, conf, err := ParseConfig("")
	if err != nil {
		t.Fatalf("ParseConfig error: %v", err)
	}
	if conf.authType != AuthTypeBasic {
		t.Errorf("authType = %v, want AuthTypeBasic", conf.authType)
	}
	if conf.oAuthToken == nil {
		t.Error("oAuthToken cache must be initialised by ParseConfig (prevents nil panic on OAuth)")
	}
	if conf.ScrapeURI != "http://broker:8080" || conf.Username != "monitor" || conf.Password != "secret" {
		t.Errorf("unexpected creds/uri: %+v", struct{ U, P, S string }{conf.Username, conf.Password, conf.ScrapeURI})
	}
	// defaults
	if conf.Timeout != 5*time.Second {
		t.Errorf("default Timeout = %v, want 5s", conf.Timeout)
	}
	if conf.SempPageSize != 100 {
		t.Errorf("default SempPageSize = %d, want 100", conf.SempPageSize)
	}
	if len(endpoints) != 0 {
		t.Errorf("expected no endpoints without a config file, got %v", endpoints)
	}
}

func TestParseConfigOAuthFromEnv(t *testing.T) {
	clearSolaceEnv(t)
	t.Setenv("SOLACE_SCRAPE_URI", "http://broker:8080")
	t.Setenv("SOLACE_OAUTH_TOKEN_URL", "http://idp/token")
	t.Setenv("SOLACE_OAUTH_CLIENT_ID", "cid")
	t.Setenv("SOLACE_OAUTH_CLIENT_SECRET", "csecret")
	t.Setenv("SOLACE_OAUTH_CLIENT_SCOPE", "scope")

	_, conf, err := ParseConfig("")
	if err != nil {
		t.Fatalf("ParseConfig error: %v", err)
	}
	if conf.authType != AuthTypeOAuth {
		t.Errorf("authType = %v, want AuthTypeOAuth", conf.authType)
	}
	if conf.oAuthToken == nil {
		t.Error("oAuthToken cache must be initialised for OAuth config")
	}
}

func TestParseConfigPartialOAuthFails(t *testing.T) {
	clearSolaceEnv(t)
	t.Setenv("SOLACE_SCRAPE_URI", "http://broker:8080")
	// Only some OAuth fields set: this must fail loudly rather than silently using default admin/admin basic auth.
	t.Setenv("SOLACE_OAUTH_TOKEN_URL", "http://idp/token")
	t.Setenv("SOLACE_OAUTH_CLIENT_ID", "cid")
	// missing SOLACE_OAUTH_CLIENT_SECRET and SOLACE_OAUTH_CLIENT_SCOPE

	if _, _, err := ParseConfig(""); err == nil {
		t.Fatal("expected error for partially configured OAuth, got nil")
	}
}

func TestParseConfigMissingScrapeURIFails(t *testing.T) {
	clearSolaceEnv(t)
	t.Setenv("SOLACE_USERNAME", "monitor")
	t.Setenv("SOLACE_PASSWORD", "secret")

	if _, _, err := ParseConfig(""); err == nil {
		t.Fatal("expected error when scrapeUri is missing, got nil")
	}
}

func TestParseConfigEndpointsFromIni(t *testing.T) {
	clearSolaceEnv(t)
	dir := t.TempDir()
	iniPath := filepath.Join(dir, "solace.ini")
	ini := `[solace]
scrapeUri=http://broker:8080
username=monitor
password=secret

[endpoint.std]
Health=*|*
Vpn=*|*
`
	if err := os.WriteFile(iniPath, []byte(ini), 0o600); err != nil {
		t.Fatal(err)
	}

	endpoints, conf, err := ParseConfig(iniPath)
	if err != nil {
		t.Fatalf("ParseConfig error: %v", err)
	}
	if conf.ScrapeURI != "http://broker:8080" {
		t.Errorf("ScrapeURI = %q, want http://broker:8080", conf.ScrapeURI)
	}
	ds, ok := endpoints["std"]
	if !ok {
		t.Fatalf("endpoint 'std' not parsed, got %v", endpoints)
	}
	if len(ds) != 2 {
		t.Errorf("endpoint 'std' has %d datasources, want 2 (%v)", len(ds), ds)
	}
}
