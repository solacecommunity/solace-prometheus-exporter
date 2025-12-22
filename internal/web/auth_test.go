package web

import (
	"net/http"
	"net/http/httptest"
	"solace_exporter/internal/exporter"
	"strings"
	"testing"
)

func newTestHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
}

func TestWrapWithAuthNoAuthRequired(t *testing.T) {
	t.Parallel()

	handler := newTestHandler()

	authConf := exporter.ExporterAuthConfig{
		Scheme:   "none",
		Username: "",
		Password: "",
	}

	wrapped := WrapWithAuth(handler, authConf)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	if strings.TrimSpace(rr.Body.String()) != "ok" {
		t.Errorf("expected body 'ok', got '%s'", rr.Body.String())
	}
}

func TestWrapWithAuthBasicAuthSuccess(t *testing.T) {
	t.Parallel()

	handler := newTestHandler()

	authConf := exporter.ExporterAuthConfig{
		Scheme:   "basic",
		Username: "admin",
		Password: "secret",
	}

	wrapped := WrapWithAuth(handler, authConf)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("admin", "secret")

	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	if strings.TrimSpace(rr.Body.String()) != "ok" {
		t.Errorf("expected body 'ok', got '%s'", rr.Body.String())
	}
}

func TestWrapWithAuthBasicAuthFailure(t *testing.T) {
	t.Parallel()

	handler := newTestHandler()

	authConf := exporter.ExporterAuthConfig{
		Scheme:   "basic",
		Username: "admin",
		Password: "secret",
	}

	wrapped := WrapWithAuth(handler, authConf)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("wrong", "creds")

	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}

	if rr.Header().Get("WWW-Authenticate") == "" {
		t.Errorf("expected WWW-Authenticate header to be set")
	}

	if !strings.Contains(rr.Body.String(), "unauthorized") {
		t.Errorf("expected body to contain 'unauthorized', got '%s'", rr.Body.String())
	}
}
