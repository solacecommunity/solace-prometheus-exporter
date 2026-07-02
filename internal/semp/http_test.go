package semp

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func newHTTPTestSemp(t *testing.T, status int, body string) *Semp {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	return NewSemp(logger, server.URL, http.Client{}, nil, false, false)
}

func TestPostHTTPSuccess(t *testing.T) {
	t.Parallel()
	s := newHTTPTestSemp(t, http.StatusOK, "<ok/>")
	rc, err := s.postHTTP(s.brokerURI+"/SEMP", "application/xml", "<rpc/>", "Test", 1)
	if err != nil {
		t.Fatalf("postHTTP error: %v", err)
	}
	defer func() { _ = rc.Close() }()
	b, _ := io.ReadAll(rc)
	if string(b) != "<ok/>" {
		t.Errorf("body = %q, want <ok/>", string(b))
	}
}

func TestPostHTTPStatusErrors(t *testing.T) {
	t.Parallel()
	for _, status := range []int{http.StatusUnauthorized, http.StatusInternalServerError} {
		s := newHTTPTestSemp(t, status, "boom")
		rc, err := s.postHTTP(s.brokerURI+"/SEMP", "application/xml", "<rpc/>", "Test", 1)
		if err == nil {
			_ = rc.Close()
			t.Errorf("postHTTP status %d: expected error, got nil", status)
		}
		if rc != nil {
			t.Errorf("postHTTP status %d: expected nil body on error", status)
		}
	}
}

func TestGetHTTPbytesSuccessAndClientError(t *testing.T) {
	t.Parallel()
	// 200 -> body returned
	s := newHTTPTestSemp(t, http.StatusOK, `{"ok":true}`)
	b, err := s.getHTTPbytes(s.brokerURI, "application/json", "Test", 1)
	if err != nil {
		t.Fatalf("getHTTPbytes 200 error: %v", err)
	}
	if string(b) != `{"ok":true}` {
		t.Errorf("body = %q", string(b))
	}

	// 4xx -> body still returned (SEMP v2 returns error detail in a 400 body, which the caller parses)
	s = newHTTPTestSemp(t, http.StatusBadRequest, `{"error":"bad"}`)
	b, err = s.getHTTPbytes(s.brokerURI, "application/json", "Test", 1)
	if err != nil {
		t.Fatalf("getHTTPbytes 400 error: %v", err)
	}
	if string(b) != `{"error":"bad"}` {
		t.Errorf("body = %q", string(b))
	}
}

func TestGetHTTPbytesServerErrorReturnsError(t *testing.T) {
	t.Parallel()
	s := newHTTPTestSemp(t, http.StatusInternalServerError, "boom")
	if _, err := s.getHTTPbytes(s.brokerURI, "application/json", "Test", 1); err == nil {
		t.Error("getHTTPbytes 500: expected error, got nil")
	}
}

// TestVisitorNilDoesNotPanic ensures a nil httpRequestVisitor (e.g. when auth setup failed) does not panic the scrape.
func TestVisitorNilDoesNotPanic(t *testing.T) {
	t.Parallel()
	s := newHTTPTestSemp(t, http.StatusOK, "<ok/>") // NewSemp called with nil visitor
	rc, err := s.postHTTP(s.brokerURI+"/SEMP", "application/xml", "<rpc/>", "Test", 1)
	if err != nil {
		t.Fatalf("postHTTP with nil visitor error: %v", err)
	}
	_ = rc.Close()
}
