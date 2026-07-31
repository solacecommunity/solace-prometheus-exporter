package main

import (
	"log/slog"
	"solace_exporter/internal/secret"
	"testing"
)

// newTestResolver returns a secret.Resolver suitable for tests that exercise doHandle/resolveRequestConfig with
// plain (non-"vault:") credentials. Resolver.Resolve is a no-op passthrough for those, so it never talks to Vault
// here -- this works whether or not the test environment happens to have VAULT_TOKEN/VAULT_TOKEN_FILE set.
func newTestResolver(t *testing.T) *secret.Resolver {
	t.Helper()
	return secret.NewResolver(nil, slog.Default())
}
