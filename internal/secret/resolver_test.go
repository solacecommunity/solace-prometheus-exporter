package secret

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
)

// stubBackend is a minimal Backend for unit tests.
type stubBackend struct {
	val string
	ttl time.Duration
	err error
}

func (s *stubBackend) ReadSecret(_ context.Context, path, field string) (string, time.Duration, error) {
	return s.val, s.ttl, s.err
}

func TestIsRef(t *testing.T) {
	cases := map[string]bool{
		"vault:secret/data/solace/solace01#password": true,
		"admin":  false,
		"":       false,
		"vault:": true,
	}
	for in, want := range cases {
		if got := IsRef(in); got != want {
			t.Errorf("IsRef(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestResolvePassthrough(t *testing.T) {
	r := NewResolver(nil, slog.Default())

	got, err := r.Resolve(context.Background(), "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "admin" {
		t.Errorf("Resolve(%q) = %q, want %q", "admin", got, "admin")
	}
}

func TestResolveInvalidRef(t *testing.T) {
	r := NewResolver(nil, slog.Default())

	_, err := r.Resolve(context.Background(), "vault:secret/data/solace/solace01")
	if err == nil {
		t.Fatal("expected error for vault ref missing #field, got nil")
	}
}

func TestResolveNilBackendReturnsError(t *testing.T) {
	r := NewResolver(nil, slog.Default())

	got, err := r.Resolve(context.Background(), "vault:secret/data/solace/solace01#password")
	if err == nil {
		t.Fatal("expected error for vault ref with nil backend, got nil")
	}
	if got != "" {
		t.Errorf("expected empty value on error, got %q", got)
	}
}

func TestResolveVaultRefEmptyBackend(t *testing.T) {
	r := NewResolver(nil, slog.Default())

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid ref no backend", "vault:secret/data/solace/solace01#password", true},
		{"valid ref short path", "vault:a#b", true},
		{"missing field separator", "vault:onlypath", true},
		{"empty path and field", "vault:#field", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := r.Resolve(context.Background(), tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestResolveCacheHit(t *testing.T) {
	r := NewResolver(nil, slog.Default())

	key := "secret/data/solace/solace01#password"
	r.cache[key] = cachedSecret{
		value:     "cached-secret",
		expiresAt: time.Now().Add(time.Minute),
	}

	got, err := r.Resolve(context.Background(), "vault:"+key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "cached-secret" {
		t.Errorf("Resolve() = %q, want %q", got, "cached-secret")
	}
}

func TestResolveCacheExpired(t *testing.T) {
	r := NewResolver(nil, slog.Default())

	key := "secret/data/solace/solace01#password"
	r.cache[key] = cachedSecret{
		value:     "stale-secret",
		expiresAt: time.Now().Add(-time.Second),
	}

	_, err := r.Resolve(context.Background(), "vault:"+key)
	if err == nil {
		t.Fatal("expected error for expired cache with nil backend, got nil")
	}
}

func TestResolveBackendCalled(t *testing.T) {
	backend := &stubBackend{val: "from-backend", ttl: time.Minute}
	r := NewResolver(backend, slog.Default())

	got, err := r.Resolve(context.Background(), "vault:secret/data/solace/solace01#password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "from-backend" {
		t.Errorf("Resolve() = %q, want %q", got, "from-backend")
	}
}

func TestResolveBackendError(t *testing.T) {
	backend := &stubBackend{err: fmt.Errorf("connection refused")}
	r := NewResolver(backend, slog.Default())

	_, err := r.Resolve(context.Background(), "vault:secret/data/solace/solace01#password")
	if err == nil {
		t.Fatal("expected error from backend, got nil")
	}
}

func TestResolveBackendCacheExpiry(t *testing.T) {
	callCount := 0
	backend := &countingBackend{
		val: "fresh",
		ttl: time.Millisecond,
		fn:  func() { callCount++ },
	}
	r := NewResolver(backend, slog.Default())

	ctx := context.Background()
	if _, err := r.Resolve(ctx, "vault:s#f"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := r.Resolve(ctx, "vault:s#f"); err != nil { // cached
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 call, got %d", callCount)
	}

	time.Sleep(5 * time.Millisecond)
	if _, err := r.Resolve(ctx, "vault:s#f"); err != nil { // expired, should call again
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 calls after expiry, got %d", callCount)
	}
}

type countingBackend struct {
	val string
	ttl time.Duration
	fn  func()
}

func (c *countingBackend) ReadSecret(_ context.Context, path, field string) (string, time.Duration, error) {
	c.fn()
	return c.val, c.ttl, nil
}

func TestHasVaultEnv(t *testing.T) {
	keys := []string{"VAULT_ADDR", "VAULT_TOKEN", "VAULT_TOKEN_FILE", "VAULT_CACERT", "VAULT_SKIP_VERIFY"}
	saved := make(map[string]string, len(keys))
	for _, k := range keys {
		saved[k] = os.Getenv(k)
		_ = os.Unsetenv(k)
	}
	defer func() {
		for _, k := range keys {
			if v, ok := saved[k]; ok {
				_ = os.Setenv(k, v)
			} else {
				_ = os.Unsetenv(k)
			}
		}
	}()

	if hasVaultEnv() {
		t.Fatal("hasVaultEnv() = true with all VAULT_* vars unset")
	}

	_ = os.Setenv("VAULT_ADDR", "https://vault.example.com:8200")

	if !hasVaultEnv() {
		t.Fatal("hasVaultEnv() = false after setting VAULT_ADDR")
	}
}

func TestParseTokenRenewability(t *testing.T) {
	tests := []struct {
		name          string
		data          map[string]interface{}
		wantRenewable bool
		wantTTL       int
	}{
		{
			name:          "renewable with json.Number ttl",
			data:          map[string]interface{}{"renewable": true, "ttl": json.Number("3600")},
			wantRenewable: true,
			wantTTL:       3600,
		},
		{
			name:          "renewable with float64 ttl",
			data:          map[string]interface{}{"renewable": true, "ttl": float64(1800)},
			wantRenewable: true,
			wantTTL:       1800,
		},
		{
			name:          "root token: not renewable, no ttl",
			data:          map[string]interface{}{"renewable": false, "ttl": json.Number("0")},
			wantRenewable: false,
			wantTTL:       0,
		},
		{
			name:          "not renewable but has a ttl (a fixed-TTL non-renewable token)",
			data:          map[string]interface{}{"renewable": false, "ttl": json.Number("600")},
			wantRenewable: false,
			wantTTL:       600,
		},
		{
			name:          "missing fields entirely",
			data:          map[string]interface{}{},
			wantRenewable: false,
			wantTTL:       0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renewable, ttl := parseTokenRenewability(tt.data)
			if renewable != tt.wantRenewable {
				t.Errorf("renewable = %v, want %v", renewable, tt.wantRenewable)
			}
			if ttl != tt.wantTTL {
				t.Errorf("ttlSeconds = %d, want %d", ttl, tt.wantTTL)
			}
		})
	}
}

func TestNextRenewalRetryDelay(t *testing.T) {
	tests := []struct {
		name                string
		consecutiveFailures int
		want                time.Duration
	}{
		{"zero treated as first failure", 0, renewalRetryBaseDelay},
		{"first failure", 1, renewalRetryBaseDelay},
		{"second failure doubles", 2, 2 * renewalRetryBaseDelay},
		{"third failure doubles again", 3, 4 * renewalRetryBaseDelay},
		{"negative treated as first failure", -5, renewalRetryBaseDelay},
		{"caps at renewalRetryMaxDelay", 10, renewalRetryMaxDelay},
		{"stays capped for very large failure counts", 100_000, renewalRetryMaxDelay},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nextRenewalRetryDelay(tt.consecutiveFailures)
			if got != tt.want {
				t.Errorf("nextRenewalRetryDelay(%d) = %v, want %v", tt.consecutiveFailures, got, tt.want)
			}
		})
	}
}

func TestApplyJitter(t *testing.T) {
	const base = 100 * time.Second

	tests := []struct {
		name string
		rnd  float64
		want time.Duration
	}{
		{"draw of 0 gives the low end", 0, 80 * time.Second},
		{"draw of 0.5 leaves the delay unchanged", 0.5, base},
		{"draw of ~1 gives the high end", 1, 120 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyJitter(base, func() float64 { return tt.rnd })
			if got != tt.want {
				t.Errorf("applyJitter(%v, %v) = %v, want %v", base, tt.rnd, got, tt.want)
			}
		})
	}
}

// TestApplyJitterStaysInBounds checks the real rand source never escapes ±renewalRetryJitter, so a jittered delay
// can't collapse to zero (a hot retry loop) or blow past the cap.
func TestApplyJitterStaysInBounds(t *testing.T) {
	d := nextRenewalRetryDelay(1)
	low := time.Duration(float64(d) * (1 - renewalRetryJitter))
	high := time.Duration(float64(d) * (1 + renewalRetryJitter))

	for i := 0; i < 1000; i++ {
		got := applyJitter(d, rand.Float64)
		if got < low || got > high {
			t.Fatalf("applyJitter(%v) = %v, want within [%v, %v]", d, got, low, high)
		}
	}
}

// TestStartTokenRenewalExitsOnContextCancel is the regression test for the renewal loop being un-cancelable: it
// used to run under context.Background() for the life of the process.
func TestStartTokenRenewalExitsOnContextCancel(t *testing.T) {
	// Port 1 refuses immediately, so the first RenewSelf attempt fails fast instead of waiting on a real dial.
	t.Setenv("VAULT_ADDR", "http://127.0.0.1:1")
	t.Setenv("VAULT_TOKEN", "test-token")

	cfg := vaultapi.DefaultConfig()
	cfg.MaxRetries = 0 // don't spend the vault client's default retry budget on a port we know is closed

	client, err := vaultapi.NewClient(cfg)
	if err != nil {
		t.Skipf("vault/api client not available: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := startTokenRenewal(ctx, client, slog.New(slog.NewTextHandler(io.Discard, nil)))

	cancel()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("renewal loop still running after context cancel")
	}
}

func TestAuthenticateTokenEnv(t *testing.T) {
	_ = os.Setenv("VAULT_TOKEN", "test-token")
	defer func() { _ = os.Unsetenv("VAULT_TOKEN") }()

	logger := slog.Default()
	client, err := vaultapi.NewClient(vaultapi.DefaultConfig())
	if err != nil {
		t.Skipf("vault/api client not available: %v", err)
	}

	authed, err := authenticate(client, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !authed {
		t.Fatal("expected authenticated = true")
	}
}

func TestAuthenticateTokenFile(t *testing.T) {
	dir := t.TempDir()
	tokenFile := filepath.Join(dir, "vault-token")
	if err := os.WriteFile(tokenFile, []byte("file-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_ = os.Setenv("VAULT_TOKEN_FILE", tokenFile)
	defer func() { _ = os.Unsetenv("VAULT_TOKEN_FILE") }()

	logger := slog.Default()
	client, err := vaultapi.NewClient(vaultapi.DefaultConfig())
	if err != nil {
		t.Skipf("vault/api client not available: %v", err)
	}

	authed, err := authenticate(client, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !authed {
		t.Fatal("expected authenticated = true")
	}

	if _, statErr := os.Stat(tokenFile); !os.IsNotExist(statErr) {
		t.Errorf("token file %q was not deleted after reading", tokenFile)
	}
}

func TestAuthenticateNeitherSet(t *testing.T) {
	_ = os.Unsetenv("VAULT_TOKEN")
	_ = os.Unsetenv("VAULT_TOKEN_FILE")

	logger := slog.Default()
	client, err := vaultapi.NewClient(vaultapi.DefaultConfig())
	if err != nil {
		t.Skipf("vault/api client not available: %v", err)
	}

	authed, err := authenticate(client, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if authed {
		t.Fatal("expected authenticated = false when no token env set")
	}
}

func TestAuthenticateTokenFileMissing(t *testing.T) {
	_ = os.Setenv("VAULT_TOKEN_FILE", "/nonexistent/path/vault-token")
	defer func() { _ = os.Unsetenv("VAULT_TOKEN_FILE") }()

	logger := slog.Default()
	client, err := vaultapi.NewClient(vaultapi.DefaultConfig())
	if err != nil {
		t.Skipf("vault/api client not available: %v", err)
	}

	_, err = authenticate(client, logger)
	if err == nil {
		t.Fatal("expected error for missing VAULT_TOKEN_FILE")
	}
}

func saveAndClearEnv(keys []string) map[string]string {
	saved := make(map[string]string, len(keys))
	for _, k := range keys {
		saved[k] = os.Getenv(k)
		_ = os.Unsetenv(k)
	}
	return saved
}

func restoreEnv(saved map[string]string, keys []string) {
	for _, k := range keys {
		if v, ok := saved[k]; ok {
			_ = os.Setenv(k, v)
		} else {
			_ = os.Unsetenv(k)
		}
	}
}

func TestNewResolverFromConfigEmpty(t *testing.T) {
	r, err := NewResolverFromConfig(context.TODO(), "", time.Minute, slog.Default())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.backend != nil {
		t.Fatal("expected nil backend for empty SECRET_BACKEND")
	}
	got, err := r.Resolve(context.Background(), "plaintext-secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "plaintext-secret" {
		t.Errorf("Resolve() = %q, want %q", got, "plaintext-secret")
	}
}

func TestNewResolverFromConfigNone(t *testing.T) {
	r, err := NewResolverFromConfig(context.TODO(), "none", time.Minute, slog.Default())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.backend != nil {
		t.Fatal("expected nil backend for SECRET_BACKEND=none")
	}
}

func TestNewResolverFromConfigUnknown(t *testing.T) {
	_, err := NewResolverFromConfig(context.TODO(), "aws-secrets-manager", time.Minute, slog.Default())
	if err == nil {
		t.Fatal("expected error for unknown SECRET_BACKEND")
	}
}

func TestNewResolverFromConfigHashicorpNoVaultEnv(t *testing.T) {
	vaultKeys := []string{"VAULT_ADDR", "VAULT_TOKEN", "VAULT_TOKEN_FILE", "VAULT_CACERT", "VAULT_SKIP_VERIFY"}
	saved := saveAndClearEnv(vaultKeys)
	defer restoreEnv(saved, vaultKeys)

	r, err := NewResolverFromConfig(context.TODO(), "hashicorp", time.Minute, slog.Default())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.backend != nil {
		t.Fatal("expected nil backend when SECRET_BACKEND=hashicorp but no VAULT_* env vars set")
	}
}
