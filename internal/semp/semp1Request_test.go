package semp

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"solace_exporter/internal/semp/types"
)

type testReply struct {
	RPC struct {
		Value []string `xml:"show>value"`
	} `xml:"rpc"`
	types.Reply
}

func testReplyXML(values string, moreCookie string, result string) string {
	return `<rpc-reply><rpc><show>` + values + `</show></rpc>` + moreCookie +
		`<execute-result code="` + result + `" reason="r"/></rpc-reply>`
}

func TestFetchSemp1DecodesReplyWithEmbeddedEnvelope(t *testing.T) {
	t.Parallel()
	s, fb := newFakeBroker(t, false, func(_, _ string) (int, string) {
		return http.StatusOK, testReplyXML("<value>a</value><value>b</value>", "<more-cookie><rpc>next</rpc></more-cookie>", "ok")
	})

	var target testReply
	up, err := s.fetchSemp1("<rpc>cmd</rpc>", "Test", 1, &target)

	if up != 1 || err != nil {
		t.Fatalf("got (%v, %v), want (1, nil)", up, err)
	}
	if strings.Join(target.RPC.Value, ",") != "a,b" {
		t.Errorf("values = %v, want [a b]", target.RPC.Value)
	}
	if target.MoreCookie.RPC != "<rpc>next</rpc>" {
		t.Errorf("more-cookie = %q, want <rpc>next</rpc>", target.MoreCookie.RPC)
	}
	if len(fb.requests) != 1 || fb.requests[0] != "POST /SEMP <rpc>cmd</rpc>" {
		t.Errorf("requests = %q", fb.requests)
	}
}

func TestFetchSemp1ReturnContract(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		status  int
		body    string
		wantUp  float64
		wantErr string
	}{
		{"broker error is unrecoverable", http.StatusUnauthorized, "denied", -1, "HTTP status 401 (Unauthorized)"},
		{"malformed reply is recoverable", http.StatusOK, "<rpc-reply>", 0, "XML syntax error on line 1: unexpected EOF"},
		{"failed execute-result is recoverable", http.StatusOK, testReplyXML("", "", "fail"), 0, "unexpected result: r"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s, _ := newFakeBroker(t, false, func(_, _ string) (int, string) { return tt.status, tt.body })

			up, err := s.fetchSemp1("<rpc/>", "Test", 1, &testReply{})

			if up != tt.wantUp {
				t.Errorf("up = %v, want %v", up, tt.wantUp)
			}
			if err == nil || err.Error() != tt.wantErr {
				t.Errorf("err = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestFetchSemp1FailedResultIsTyped(t *testing.T) {
	t.Parallel()
	s, _ := newFakeBroker(t, false, func(_, _ string) (int, string) {
		return http.StatusOK, testReplyXML("", "", "fail")
	})

	_, err := s.fetchSemp1("<rpc/>", "Test", 1, &testReply{})

	var resultErr *unexpectedResultError
	if !errors.As(err, &resultErr) || resultErr.reason != "r" {
		t.Errorf("err = %#v, want *unexpectedResultError with reason r", err)
	}
}

func TestFetchSemp1PagesFollowsMoreCookie(t *testing.T) {
	t.Parallel()
	s, fb := newFakeBroker(t, false, func(_, body string) (int, string) {
		switch body {
		case "<rpc>first</rpc>":
			return http.StatusOK, testReplyXML("<value>1</value>", "<more-cookie><rpc>second</rpc></more-cookie>", "ok")
		case "<rpc>second</rpc>":
			return http.StatusOK, testReplyXML("<value>2</value>", "", "ok")
		}
		return http.StatusBadRequest, ""
	})

	var seen []string
	up, err := fetchSemp1Pages(s, "<rpc>first</rpc>", "Test", func(page *testReply) {
		seen = append(seen, page.RPC.Value...)
	})

	if up != 1 || err != nil {
		t.Fatalf("got (%v, %v), want (1, nil)", up, err)
	}
	if strings.Join(seen, ",") != "1,2" {
		t.Errorf("handled values = %v, want [1 2]", seen)
	}
	if len(fb.requests) != 2 {
		t.Errorf("requests = %q, want 2", fb.requests)
	}
}

func TestFetchSemp1PagesStopsOnFailingPageAndKeepsEarlierPages(t *testing.T) {
	t.Parallel()
	s, _ := newFakeBroker(t, false, func(_, body string) (int, string) {
		if body == "<rpc>first</rpc>" {
			return http.StatusOK, testReplyXML("<value>1</value>", "<more-cookie><rpc>second</rpc></more-cookie>", "ok")
		}
		return http.StatusInternalServerError, ""
	})

	var seen []string
	up, err := fetchSemp1Pages(s, "<rpc>first</rpc>", "Test", func(page *testReply) {
		seen = append(seen, page.RPC.Value...)
	})

	if up != -1 || err == nil {
		t.Errorf("got (%v, %v), want (-1, error)", up, err)
	}
	if strings.Join(seen, ",") != "1" {
		t.Errorf("handled values = %v, want [1]", seen)
	}
}

func TestWithLegacyResultErrorTextOnlyRewritesFailedResults(t *testing.T) {
	t.Parallel()
	if got := withLegacyResultErrorText(&unexpectedResultError{reason: "r"}).Error(); got != "unexpected result: r. see log for further details" {
		t.Errorf("failed result: got %q", got)
	}
	other := errors.New("HTTP status 500")
	if got := withLegacyResultErrorText(other); got != other {
		t.Errorf("other error: got %v, want it unchanged", got)
	}
}
