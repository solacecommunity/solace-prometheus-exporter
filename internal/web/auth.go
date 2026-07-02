package web

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"solace_exporter/internal/exporter"
)

func WrapWithAuth(handler http.Handler, authConf exporter.ExporterAuthConfig) http.Handler {
	if (authConf.Scheme == "basic") && (len(authConf.Username) > 0) && (len(authConf.Password) > 0) {
		return basicAuth(handler, authConf)
	}
	return handler
}

func basicAuth(h http.Handler, authConf exporter.ExporterAuthConfig) http.Handler {
	// Compare fixed-size SHA-256 digests rather than the raw credentials. subtle.ConstantTimeCompare returns
	// early when the two slices differ in length, so comparing the raw values would leak credential length via
	// timing. Hashing first makes both operands a constant 32 bytes, so the comparison is constant-time with
	// respect to both the content and the length of the supplied credentials.
	wantUser := sha256.Sum256([]byte(authConf.Username))
	wantPass := sha256.Sum256([]byte(authConf.Password))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		gotUser := sha256.Sum256([]byte(u))
		gotPass := sha256.Sum256([]byte(p))
		userOK := subtle.ConstantTimeCompare(gotUser[:], wantUser[:]) == 1
		passOK := subtle.ConstantTimeCompare(gotPass[:], wantPass[:]) == 1
		if !ok || !userOK || !passOK {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted"`)
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("unauthorized\n"))
			return
		}

		h.ServeHTTP(w, r)
	})
}
