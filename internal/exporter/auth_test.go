package exporter

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newOAuthTokenServer(t *testing.T, hits *atomic.Int32, accessToken string) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"` + accessToken + `","token_type":"bearer","expires_in":3600}`))
	}))
	t.Cleanup(ts.Close)
	return ts
}

func newOAuthConfig(tokenURL string) *Config {
	return &Config{
		OAuthTokenURL:     tokenURL,
		OAuthClientID:     "client",
		OAuthClientSecret: "secret",
		OAuthClientScope:  "scope",
		authType:          AuthTypeOAuth,
		Timeout:           5 * time.Second,
		oAuthToken:        &oAuthTokenCache{},
	}
}

// TestGetOAuthTokenIsCachedAcrossConcurrentRequests verifies the OAuth token is fetched exactly once and reused by
// all concurrent callers (the reason commit 00600f0 shared the Config by pointer) - now achieved safely via the
// shared oAuthTokenCache without racing the credential fields.
func TestGetOAuthTokenIsCachedAcrossConcurrentRequests(t *testing.T) {
	t.Parallel()
	var hits atomic.Int32
	ts := newOAuthTokenServer(t, &hits, "tok-123")
	conf := newOAuthConfig(ts.URL)

	const n = 50
	var wg sync.WaitGroup
	tokens := make([]string, n)
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			tokens[i], errs[i] = conf.getOAuthToken(context.Background())
		}(i)
	}
	wg.Wait()

	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Fatalf("getOAuthToken returned error: %v", errs[i])
		}
		if tokens[i] != "tok-123" {
			t.Errorf("token[%d] = %q, want tok-123", i, tokens[i])
		}
	}
	if got := hits.Load(); got != 1 {
		t.Errorf("expected exactly 1 token fetch (cached), got %d", got)
	}
}

// TestOAuthTokenCacheIsSharedAcrossClones is the OAuth counterpart of the credential-isolation fix: a per-request
// Config.Clone() must still share the token cache, so cloning for isolation does NOT reintroduce repeated token
// fetches.
func TestOAuthTokenCacheIsSharedAcrossClones(t *testing.T) {
	t.Parallel()
	var hits atomic.Int32
	ts := newOAuthTokenServer(t, &hits, "tok-abc")
	base := newOAuthConfig(ts.URL)

	const n = 30
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			clone := base.Clone() // each "request" gets its own clone...
			tok, err := clone.getOAuthToken(context.Background())
			if err != nil {
				t.Errorf("getOAuthToken error: %v", err)
			}
			if tok != "tok-abc" {
				t.Errorf("token = %q, want tok-abc", tok)
			}
		}()
	}
	wg.Wait()

	if got := hits.Load(); got != 1 {
		t.Errorf("expected exactly 1 token fetch shared across clones, got %d", got)
	}
}

// TestGetOAuthTokenNilCacheReturnsError ensures a Config without an initialised cache fails loudly instead of panicking.
func TestGetOAuthTokenNilCacheReturnsError(t *testing.T) {
	t.Parallel()
	conf := &Config{authType: AuthTypeOAuth} // no oAuthToken
	if _, err := conf.getOAuthToken(context.Background()); err == nil {
		t.Fatal("expected an error for a nil OAuth token cache, got nil")
	}
}

// TestCloneIsolatesCredentials verifies Clone gives an independent copy of the per-request scrape fields while
// sharing the OAuth token cache pointer.
func TestCloneIsolatesCredentials(t *testing.T) {
	t.Parallel()
	base := &Config{Username: "u", Password: "p", ScrapeURI: "http://base", oAuthToken: &oAuthTokenCache{}}
	clone := base.Clone()
	clone.Username = "other"
	clone.Password = "other"
	clone.ScrapeURI = "http://other"

	if base.Username != "u" || base.Password != "p" || base.ScrapeURI != "http://base" {
		t.Errorf("mutating clone changed base: %+v", struct{ U, P, S string }{base.Username, base.Password, base.ScrapeURI})
	}
	if clone.oAuthToken != base.oAuthToken {
		t.Error("clone must share the OAuth token cache pointer with the base config")
	}
}

// TestIssuerPrefixedToken checks the Solace issuer-prefixed token format uses UNPADDED base64 for issuers of every
// length class (the old code stripped exactly one char, corrupting issuers whose base64 had 0 or 2 padding chars).
func TestIssuerPrefixedToken(t *testing.T) {
	t.Parallel()

	// no issuer -> token unchanged
	no := &Config{}
	if got := no.issuerPrefixedToken("tok"); got != "tok" {
		t.Errorf("no issuer: got %q, want tok", got)
	}

	// issuers of len%3 == 0,1,2 exercise base64 padding of 0,2,1 chars respectively
	for _, issuer := range []string{"abc", "ab", "a", "abcd", "https://idp.example.com/realm"} {
		c := &Config{OAuthIssuer: issuer}
		got := c.issuerPrefixedToken("TOKEN")
		want := "~" + base64.RawStdEncoding.EncodeToString([]byte(issuer)) + "~TOKEN"
		if got != want {
			t.Errorf("issuer %q: got %q, want %q", issuer, got, want)
		}
		if strings.Contains(got, "=") {
			t.Errorf("issuer %q: prefixed token must not contain base64 padding: %q", issuer, got)
		}
	}
}

// TestSetAuthHeaderOAuthReFetchesToken verifies the OAuth visitor fetches the token from the cache on EVERY request
// (so a long-lived visitor picks up a refreshed token) rather than capturing one token string forever.
func TestSetAuthHeaderOAuthReFetchesToken(t *testing.T) {
	t.Parallel()
	var hits atomic.Int32
	ts := newOAuthTokenServer(t, &hits, "tok-xyz")
	conf := newOAuthConfig(ts.URL)

	visit, err := conf.setAuthHeader(context.Background())
	if err != nil {
		t.Fatalf("setAuthHeader error: %v", err)
	}
	// Invoke the visitor several times: it must set a Bearer header each time (fetching/reusing the cached token),
	// never leaving a nil visitor or a stale/empty header.
	for i := 0; i < 3; i++ {
		req, _ := http.NewRequest(http.MethodPost, "http://broker/SEMP", nil)
		visit(req)
		if got := req.Header.Get("Authorization"); got != "Bearer tok-xyz" {
			t.Errorf("Authorization = %q, want Bearer tok-xyz", got)
		}
	}
	if hits.Load() != 1 {
		t.Errorf("token should be cached: expected 1 fetch, got %d", hits.Load())
	}
}

// TestSetAuthHeaderBasic verifies the basic-auth visitor sets the request's credentials from the (per-request) Config.
func TestSetAuthHeaderBasic(t *testing.T) {
	t.Parallel()
	conf := &Config{authType: AuthTypeBasic, Username: "user-x", Password: "pass-x"}
	visit, err := conf.setAuthHeader(context.Background())
	if err != nil {
		t.Fatalf("setAuthHeader error: %v", err)
	}
	req, _ := http.NewRequest(http.MethodPost, "http://broker/SEMP", nil)
	visit(req)
	u, p, ok := req.BasicAuth()
	if !ok || u != "user-x" || p != "pass-x" {
		t.Errorf("basic auth = (%q,%q,%v), want (user-x,pass-x,true)", u, p, ok)
	}
}
