package semp

import (
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// injectedFilter tries to break out of its XML element, add an element of its own and inject a fmt verb.
const injectedFilter = `q&1'"</name></vpn-name><evil/>%d`

// requestText decodes a SEMP v1 command, failing on malformed XML, and returns its element names and text.
func requestText(t *testing.T, command string) (elements []string, text string) {
	t.Helper()
	decoder := xml.NewDecoder(strings.NewReader(command))
	var sb strings.Builder
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return elements, sb.String()
		}
		if err != nil {
			t.Fatalf("command is not well-formed XML (%v): %s", err, command)
		}
		switch tok := token.(type) {
		case xml.StartElement:
			elements = append(elements, tok.Name.Local)
		case xml.CharData:
			sb.Write(tok)
		}
	}
}

// firstCommand runs a getter against a broker that rejects every request and returns the command it sent.
func firstCommand(t *testing.T, getter semp1Getter, f filterArgs) string {
	t.Helper()
	s, fb := newFakeBroker(t, false, func(_, _ string) (int, string) {
		return http.StatusOK, `<rpc-reply><execute-result code="fail" reason="test"/></rpc-reply>`
	})
	_, _ = getter.with(f)(s, make(chan PrometheusMetric, 100))
	if len(fb.requests) == 0 {
		t.Fatal("getter sent no request")
	}
	return strings.TrimPrefix(fb.requests[0], "POST /SEMP ")
}

// TestSemp1CommandsEscapeFilters ensures filters from the scrape request are sent to the broker as plain text: they
// can neither break out of their XML element nor be interpreted as fmt verbs.
func TestSemp1CommandsEscapeFilters(t *testing.T) {
	t.Parallel()
	for name, getter := range semp1Getters {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			usesFilters := strings.Contains(firstCommand(t, getter, goldenFilters), "*")

			elements, text := requestText(t, firstCommand(t, getter, filterArgs{vpn: injectedFilter, item: injectedFilter}))

			for _, element := range elements {
				if element == "evil" {
					t.Errorf("filter injected an XML element into the command")
				}
			}
			if usesFilters && !strings.Contains(text, injectedFilter) {
				t.Errorf("command text %q does not contain the filter %q verbatim", text, injectedFilter)
			}
		})
	}
}
