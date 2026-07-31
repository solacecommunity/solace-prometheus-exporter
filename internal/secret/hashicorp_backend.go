package secret

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"strings"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
)

const (
	// minRenewSlack avoids hammering Vault with a renewal request the
	// instant a dynamic secret's lease is issued.
	minRenewSlack = 5 * time.Second

	// tokenRenewFallbackTTL paces the next renewal after a successful RenewSelf that doesn't report a lease
	// duration. Unrelated to the per-secret cache TTL (see cacheTTL below).
	tokenRenewFallbackTTL = 60 * time.Second

	// renewalRetryBaseDelay / renewalRetryMaxDelay bound the backoff between failed renewal attempts; see
	// nextRenewalRetryDelay.
	renewalRetryBaseDelay = 30 * time.Second
	renewalRetryMaxDelay  = 10 * time.Minute

	// renewalRetryJitter spreads each retry delay by this fraction (±) so a fleet of exporters sharing one Vault
	// doesn't retry in lockstep; see applyJitter.
	renewalRetryJitter = 0.2
)

// hashicorpBackend implements Backend for HashiCorp Vault.
type hashicorpBackend struct {
	client *vaultapi.Client
	logger *slog.Logger

	// cacheTTL is how long a resolved static secret value is cached before Resolve re-reads it from Vault.
	// Leased secrets ignore this and use half their lease duration instead (see ReadSecret).
	cacheTTL time.Duration
}

// hasVaultEnv reports whether any Vault-related env var is set, i.e. the operator has opted into Vault.
func hasVaultEnv() bool {
	return os.Getenv("VAULT_ADDR") != "" ||
		os.Getenv("VAULT_TOKEN") != "" ||
		os.Getenv("VAULT_TOKEN_FILE") != "" ||
		os.Getenv("VAULT_CACERT") != "" ||
		os.Getenv("VAULT_SKIP_VERIFY") != ""
}

// newHashicorpBackend builds a Vault API client from the standard VAULT_ADDR / VAULT_CACERT / VAULT_SKIP_VERIFY
// env vars, authenticates it, and returns a ready-to-use Backend (cacheTTL: see hashicorpBackend.cacheTTL). Only
// reached via explicit SECRET_BACKEND=hashicorp, so a missing VAULT_* env var here is a real misconfig (Warn).
func newHashicorpBackend(ctx context.Context, cacheTTL time.Duration, logger *slog.Logger) (Backend, error) {
	if !hasVaultEnv() {
		logger.Warn("SECRET_BACKEND=hashicorp is set but no VAULT_* env vars are configured; " +
			"falling back to a plaintext pass-through -- any \"vault:\" reference will fail to resolve until " +
			"VAULT_ADDR and VAULT_TOKEN (or VAULT_TOKEN_FILE) are set")
		return nil, nil
	}

	client, err := vaultapi.NewClient(vaultapi.DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("secret: failed to build vault client: %w", err)
	}

	authenticated, err := authenticate(client, logger)
	if err != nil {
		return nil, fmt.Errorf("secret: vault authentication failed: %w", err)
	}

	b := &hashicorpBackend{client: client, logger: logger, cacheTTL: cacheTTL}

	if authenticated {
		maybeStartTokenRenewal(ctx, client, logger)
	}

	return b, nil
}

// maybeStartTokenRenewal checks whether the client's own token is renewable before starting a background renewal
// loop -- a non-renewable token (e.g. a root token) would just retry and log an error forever. If the lookup
// itself fails, it conservatively starts the loop anyway rather than silently losing renewal.
func maybeStartTokenRenewal(ctx context.Context, client *vaultapi.Client, logger *slog.Logger) {
	renewable, ttlSeconds, err := lookupTokenRenewability(client)
	if err != nil {
		logger.Debug("Vault token lookup-self failed; attempting background renewal anyway", "err", err)
		startTokenRenewal(ctx, client, logger)
		return
	}

	if !renewable {
		if ttlSeconds > 0 {
			logger.Info("Vault token is not renewable; background renewal disabled -- rotate it manually before it expires",
				"ttlSeconds", ttlSeconds)
		} else {
			logger.Debug("Vault token is not renewable and has no expiration (e.g. a root token); background renewal disabled")
		}
		return
	}

	startTokenRenewal(ctx, client, logger)
}

// lookupTokenRenewability reports whether the client's own token is renewable and its TTL in seconds (0 if it
// never expires). Field extraction is split out into parseTokenRenewability so it's unit-testable without Vault.
func lookupTokenRenewability(client *vaultapi.Client) (renewable bool, ttlSeconds int, err error) {
	secret, err := client.Auth().Token().LookupSelf()
	if err != nil {
		return false, 0, err
	}
	if secret == nil {
		return false, 0, fmt.Errorf("lookup-self returned no data")
	}
	renewable, ttlSeconds = parseTokenRenewability(secret.Data)
	return renewable, ttlSeconds, nil
}

// parseTokenRenewability extracts "renewable" and "ttl" from a Vault lookup-self response's Data map. Vault's
// client decodes JSON numbers as either json.Number or float64 depending on the code path, so both are handled.
func parseTokenRenewability(data map[string]interface{}) (renewable bool, ttlSeconds int) {
	renewable, _ = data["renewable"].(bool)

	switch v := data["ttl"].(type) {
	case json.Number:
		if n, convErr := v.Int64(); convErr == nil {
			ttlSeconds = int(n)
		}
	case float64:
		ttlSeconds = int(v)
	}

	return renewable, ttlSeconds
}

// ReadSecret reads a single field from a Vault KV v1 or v2 secret. It respects ctx's deadline/cancellation via
// Vault's ReadWithContext, so a hung or slow Vault server can't block the caller past its own timeout.
func (b *hashicorpBackend) ReadSecret(ctx context.Context, path, field string) (string, time.Duration, error) {
	if b.client == nil {
		return "", 0, fmt.Errorf("vault client not initialized")
	}

	secret, err := b.client.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return "", 0, err
	}
	if secret == nil {
		return "", 0, fmt.Errorf("no secret found at path %q", path)
	}

	// KV v2 wraps fields one level deeper under "data" and includes a "metadata" key; KV v1 doesn't.
	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok || secret.Data["metadata"] == nil {
		data = secret.Data
	}

	raw, ok := data[field]
	if !ok {
		return "", 0, fmt.Errorf("field %q not found at path %q", field, path)
	}
	val, ok := raw.(string)
	if !ok {
		return "", 0, fmt.Errorf("field %q at path %q is not a string", field, path)
	}

	ttl := b.cacheTTL
	if secret.LeaseDuration > 0 {
		ttl = time.Duration(secret.LeaseDuration) * time.Second / 2
	}

	return val, ttl, nil
}

// authenticate sets the client's token from VAULT_TOKEN, or else a one-shot token file deleted after reading
// (e.g. an init container dropping a wrapped token into a tmpfs emptyDir). Returns (false, nil) when neither is
// set; errors only if VAULT_TOKEN_FILE was set but unreadable.
func authenticate(client *vaultapi.Client, logger *slog.Logger) (bool, error) {
	if token := os.Getenv("VAULT_TOKEN"); token != "" {
		client.SetToken(token)
		logger.Info("Vault client authenticated from VAULT_TOKEN env var")
		return true, nil
	}

	tokenFile := os.Getenv("VAULT_TOKEN_FILE")
	if tokenFile == "" {
		return false, nil
	}

	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return false, fmt.Errorf("reading VAULT_TOKEN_FILE %q: %w", tokenFile, err)
	}
	client.SetToken(strings.TrimSpace(string(data)))

	if rmErr := os.Remove(tokenFile); rmErr != nil {
		logger.Error("Failed to delete token file after reading; token may persist on disk",
			"file", tokenFile, "err", rmErr)
	} else {
		logger.Info("Vault token read from file and deleted", "file", tokenFile)
	}

	return true, nil
}

// nextRenewalRetryDelay computes the backoff before the next renewal attempt, given the number of consecutive
// failures so far (1 = first failure). It doubles from renewalRetryBaseDelay up to renewalRetryMaxDelay. Split
// out as a pure function so the backoff curve is unit-testable without a live Vault server.
func nextRenewalRetryDelay(consecutiveFailures int) time.Duration {
	if consecutiveFailures < 1 {
		consecutiveFailures = 1
	}

	delay := renewalRetryBaseDelay
	for i := 1; i < consecutiveFailures && delay < renewalRetryMaxDelay; i++ {
		delay *= 2
	}
	if delay > renewalRetryMaxDelay {
		delay = renewalRetryMaxDelay
	}
	return delay
}

// applyJitter spreads d by ±renewalRetryJitter, drawing rnd from [0,1). Split out so the backoff curve itself
// stays deterministic and both halves are testable without a live Vault server.
func applyJitter(d time.Duration, rnd func() float64) time.Duration {
	spread := float64(d) * renewalRetryJitter
	return time.Duration(float64(d) - spread + 2*spread*rnd())
}

// startTokenRenewal keeps the client's token alive for the lifetime of ctx, so a long-running exporter doesn't
// start failing scrapes once the initial token TTL elapses. Requires renewable=true; a non-renewable token just
// fails and retries with jittered backoff (see nextRenewalRetryDelay) -- ops should fix the token policy. The
// returned channel is closed once the loop has exited, so callers and tests can observe shutdown.
func startTokenRenewal(ctx context.Context, client *vaultapi.Client, logger *slog.Logger) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)

		consecutiveFailures := 0
		for {
			secret, err := client.Auth().Token().RenewSelf(0)
			if err != nil {
				consecutiveFailures++
				delay := applyJitter(nextRenewalRetryDelay(consecutiveFailures), rand.Float64)
				logger.Error("Vault token renewal failed, will retry",
					"err", err, "retryIn", delay, "consecutiveFailures", consecutiveFailures)
				select {
				case <-ctx.Done():
					return
				case <-time.After(delay):
					continue
				}
			}
			consecutiveFailures = 0

			// RenewSelf returns (nil, nil) for an empty response body, and a panic in this goroutine would take the
			// whole exporter down -- so guard secret and fall back to the default pacing.
			ttl := tokenRenewFallbackTTL
			if secret != nil && secret.Auth != nil && secret.Auth.LeaseDuration > 0 {
				ttl = time.Duration(secret.Auth.LeaseDuration) * time.Second
			}

			sleep := ttl / 2
			if sleep < minRenewSlack {
				sleep = minRenewSlack
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(sleep):
			}
		}
	}()

	return done
}
