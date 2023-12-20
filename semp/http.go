package semp

import (
	"fmt"
	"github.com/go-kit/kit/log/level"
	"io"
	"net/http"
	"strings"
	"time"
)

// 1sec
const longQuery time.Duration = 1 * 1000 * 1000 * 1000
const longQueryFirstSempV2 time.Duration = 10 * 1000 * 1000 * 1000

// Call http post for the supplied uri and body
func (s *Semp) postHTTP(uri string, _ string, body string, logName string, page int) (io.ReadCloser, error) {
	start := time.Now()

	req, err := http.NewRequest("POST", uri, strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	s.httpRequestVisitor(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	var queryDuration = time.Since(start)
	if queryDuration > longQuery {
		_ = level.Warn(s.logger).Log("msg", "Scraped "+logName+" but this took very long. Please add more cpu to your broker. Otherwise you are about to harm your broker.", "page", page, "duration", queryDuration)
	}
	_ = level.Debug(s.logger).Log("msg", "Scraped "+logName, "page", page, "duration", queryDuration, "request", body)

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("HTTP status %d (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return resp.Body, nil
}

func (s *Semp) getHTTPbytes(uri string, _ string, logName string, page int) ([]byte, error) {
	start := time.Now()

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	s.httpRequestVisitor(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	var queryDuration = time.Since(start)
	if (page > 1 && queryDuration > longQuery) || (page == 1 && queryDuration > longQueryFirstSempV2) {
		_ = level.Warn(s.logger).Log("msg", "Scraped "+logName+" but this took very long. Please add more cpu to your broker. Otherwise you are about to harm your broker.", "page", page, "duration", queryDuration)
	}
	_ = level.Debug(s.logger).Log("msg", "Scraped "+logName, "page", page, "duration", queryDuration)

	if !(resp.StatusCode >= 200 && resp.StatusCode < 500) {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("HTTP status %d (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
