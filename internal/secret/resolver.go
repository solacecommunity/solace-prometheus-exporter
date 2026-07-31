package secret

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

const vaultPrefix = "vault:"

// backendFetchTimeout bounds a shared (singleflight-coalesced) backend fetch, independent of any single caller's ctx.
const backendFetchTimeout = 30 * time.Second

// Backend is the interface secret-manager implementations must satisfy: fetch a single field from a secret at
// path, returning the value and a recommended cache TTL. Adding a new manager (AWS Secrets Manager, Azure Key
// Vault, ...) means adding a new Backend implementation -- no changes to Resolver are needed.
type Backend interface {
	// ReadSecret fetches the value of field from the secret at path. It returns the resolved value, a cache TTL
	// (how long the caller may reuse the value before re-fetching), and an error on failure. Implementations
	// should respect ctx cancellation/deadline so a slow or unreachable backend can't hang its caller forever.
	ReadSecret(ctx context.Context, path, field string) (value string, cacheTTL time.Duration, err error)
}

// cachedSecret holds a previously resolved secret value and its expiry.
type cachedSecret struct {
	value     string
	expiresAt time.Time
}

// Resolver resolves "<prefix><path>#<field>" references to live secret values using a pluggable Backend; any
// other string is returned unchanged. Safe for concurrent use -- concurrent resolves of the same ref are
// collapsed into one backend call (sf), so a slow backend only stalls callers waiting on that ref.
type Resolver struct {
	backend Backend
	logger  *slog.Logger

	mu    sync.Mutex
	cache map[string]cachedSecret

	sf singleflight.Group
}

// NewResolver creates a Resolver backed by the given Backend. If backend is nil, it's a pass-through that only
// errors on "vault:"-prefixed refs -- useful when no secret manager is configured.
func NewResolver(backend Backend, logger *slog.Logger) *Resolver {
	return &Resolver{
		backend: backend,
		logger:  logger,
		cache:   map[string]cachedSecret{},
	}
}

// NewResolverFromConfig builds a Resolver for backendType ("hashicorp" for Vault, "" / "none" for a no-op
// pass-through). cacheTTL controls how long a resolved static secret is cached (0 disables it); leased secrets
// ignore this and use half their lease duration instead.
func NewResolverFromConfig(ctx context.Context, backendType string, cacheTTL time.Duration, logger *slog.Logger) (*Resolver, error) {
	switch strings.ToLower(backendType) {
	case "", "none":
		logger.Debug("SECRET_BACKEND not set; secret resolver is a no-op pass-through")
		return NewResolver(nil, logger), nil
	case "hashicorp":
		b, err := newHashicorpBackend(ctx, cacheTTL, logger)
		if err != nil {
			return nil, fmt.Errorf("secret: vault backend: %w", err)
		}
		return NewResolver(b, logger), nil
	default:
		return nil, fmt.Errorf("secret: unknown SECRET_BACKEND %q; supported values: hashicorp", backendType)
	}
}

// Resolve returns the literal value for cfg. A "<prefix><path>#<field>" reference is read from the backend
// (cached in-memory); any other string is returned unchanged. ctx bounds how long this caller waits; concurrent
// resolves of the same ref share one backend call bounded by its own timeout, not any single caller's ctx.
func (r *Resolver) Resolve(ctx context.Context, cfg string) (string, error) {
	ref, hasVaultPrefix := strings.CutPrefix(cfg, vaultPrefix)
	if !hasVaultPrefix {
		return cfg, nil
	}

	path, field, ok := strings.Cut(ref, "#")
	if !ok || path == "" || field == "" {
		return "", fmt.Errorf("secret: invalid vault ref %q, expected <path>#<field>", ref)
	}

	if val, found := r.cacheGet(ref); found {
		return val, nil
	}

	if r.backend == nil {
		return "", fmt.Errorf("vault client not initialized; set environment variable VAULT_ADDR and VAULT_TOKEN (or VAULT_TOKEN_FILE)")
	}

	// DoChan (not Do) lets a caller whose own ctx is canceled return immediately via the select below, instead of
	// blocking until the shared in-flight call finishes.
	type result struct {
		val       string
		ttl       time.Duration
		fromCache bool
	}
	ch := r.sf.DoChan(ref, func() (interface{}, error) {
		// Re-check the cache: another goroutine may have populated it between our check above and acquiring the
		// singleflight slot.
		if val, found := r.cacheGet(ref); found {
			return result{val: val, fromCache: true}, nil
		}

		// This closure runs once per in-flight ref, under whichever caller happened to win the singleflight race --
		// so it must not run under that caller's ctx, or their cancellation would abort the fetch for every other
		// waiter too. WithoutCancel detaches from it; WithTimeout gives the fetch its own independent bound.
		fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), backendFetchTimeout)
		defer cancel()

		val, ttl, err := r.backend.ReadSecret(fetchCtx, path, field)
		if err != nil {
			return result{}, err
		}
		return result{val: val, ttl: ttl}, nil
	})

	select {
	case <-ctx.Done():
		return "", fmt.Errorf("secret: resolving %q: %w", ref, ctx.Err())
	case out := <-ch:
		if out.Err != nil {
			return "", fmt.Errorf("secret: resolving %q: %w", ref, out.Err)
		}
		res := out.Val.(result)
		if !res.fromCache {
			r.cacheSet(ref, res.val, res.ttl)
		}
		return res.val, nil
	}
}

func (r *Resolver) cacheGet(ref string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cached, found := r.cache[ref]
	if !found {
		return "", false
	}
	if time.Now().After(cached.expiresAt) {
		delete(r.cache, ref)
		return "", false
	}
	return cached.value, true
}

func (r *Resolver) cacheSet(ref, val string, ttl time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ttl <= 0 {
		delete(r.cache, ref)
		return
	}
	r.cache[ref] = cachedSecret{value: val, expiresAt: time.Now().Add(ttl)}
}

// IsRef reports whether cfg looks like a vault reference, without resolving it -- useful for config
// validation/logging (e.g. a --check-config dry run) that wants to say "this field is vault-backed".
func IsRef(cfg string) bool {
	return strings.HasPrefix(cfg, vaultPrefix)
}
