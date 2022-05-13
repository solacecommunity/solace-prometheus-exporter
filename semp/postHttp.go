package semp

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Call http post for the supplied uri and body
func (s *Semp) postHTTP(uri string, _ string, body string) (io.ReadCloser, error) {
	req, err := http.NewRequest("POST", uri, strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	s.httpRequestVisitor(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("HTTP status %d (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return resp.Body, nil
}
