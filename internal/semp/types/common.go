package types

import (
	"fmt"
)

type ExecuteResult struct {
	Result string `xml:"code,attr"`
	Reason string `xml:"reason,attr"`
}

func (e ExecuteResult) OK() error {
	if e.Result == "ok" {
		return nil
	}
	return fmt.Errorf("unexpected result: %s", e.Reason)
}

type MoreCookie struct {
	RPC string `xml:",innerxml"`
}

// Reply holds the elements every SEMP v1 reply shares. Embed it in a reply struct: encoding/xml promotes the fields
// of an embedded struct, so they are decoded as if declared in the reply itself.
type Reply struct {
	MoreCookie    MoreCookie    `xml:"more-cookie,omitempty"`
	ExecuteResult ExecuteResult `xml:"execute-result"`
}

// SempReply gives access to the shared reply elements of any struct embedding Reply.
func (r *Reply) SempReply() *Reply {
	return r
}
