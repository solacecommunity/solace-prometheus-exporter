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
