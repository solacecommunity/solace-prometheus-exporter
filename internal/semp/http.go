package semp

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-kit/log/level"
)

const longQuery time.Duration = 2 * 1000 * 1000 * 1000             // 2 seconds
const longQueryFirstSempV2 time.Duration = 15 * 1000 * 1000 * 1000 // 15 seconds

// Call http post for the supplied uri and body
func (semp *Semp) postHTTP(uri string, _ string, body string, logName string, page int) (io.ReadCloser, error) {
	start := time.Now()

	req, err := http.NewRequest("POST", uri, strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	semp.httpRequestVisitor(req)

	resp, err := semp.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	var queryDuration = time.Since(start)
	if queryDuration > longQuery {
		_ = level.Warn(semp.logger).Log("msg", "Scraped "+logName+" but this took very long. Please add more cpu to your broker. Otherwise you are about to harm your broker.", "page", page, "duration", queryDuration)
	}
	_ = level.Debug(semp.logger).Log("msg", "Scraped "+logName, "page", page, "duration", queryDuration)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("HTTP status %d (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return resp.Body, nil
}

func (semp *Semp) getHTTPbytes(uri string, _ string, logName string, page int) ([]byte, error) {
	start := time.Now()

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	semp.httpRequestVisitor(req)

	resp, err := semp.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	var queryDuration = time.Since(start)
	if semp.logBrokerToSlowWarnings && (page > 1 && queryDuration > longQuery) || (page == 1 && queryDuration > longQueryFirstSempV2) {
		_ = level.Warn(semp.logger).Log("msg", "Scraped "+logName+" but this took very long. Please add more cpu to your broker. Otherwise you are about to harm your broker.", "page", page, "duration", queryDuration)
	}

	_ = level.Debug(semp.logger).Log("msg", "Scraped "+logName, "page", page, "duration", queryDuration)

	if resp.StatusCode < 200 || resp.StatusCode >= 500 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("HTTP status %d (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
