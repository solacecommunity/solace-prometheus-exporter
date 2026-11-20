package semp

import (
	"encoding/xml"
	"errors"
	"solace_exporter/internal/semp/types"
)

// semp1Reply is a decoded SEMP v1 reply; reply structs satisfy it by embedding types.Reply.
type semp1Reply interface {
	SempReply() *types.Reply
}

// unexpectedResultError reports a SEMP v1 execute-result other than "ok". Its text ends up in the error label of
// solace_up, so it must stay stable.
type unexpectedResultError struct {
	reason string
}

func (e *unexpectedResultError) Error() string {
	return "unexpected result: " + e.reason
}

// withLegacyResultErrorText keeps the longer error text a few getters have always reported for a failed
// execute-result, so their solace_up error label does not change. Other errors pass through unchanged.
func withLegacyResultErrorText(err error) error {
	var resultErr *unexpectedResultError
	if errors.As(err, &resultErr) {
		return errors.New(resultErr.Error() + ". see log for further details")
	}
	return err
}

// fetchSemp1 posts a SEMP v1 command to the broker and decodes the reply into target. The returned up value follows
// the getter contract: 1 on success, -1 if the broker cannot be reached (unrecoverable, aborts the remaining
// targets), 0 if the reply cannot be decoded or reports a failed execute-result.
func (semp *Semp) fetchSemp1(command string, logName string, page int, target semp1Reply) (float64, error) {
	body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, logName, page)
	if err != nil {
		semp.logger.Error("Can't scrape "+logName, "err", err, "broker", semp.brokerURI)
		return -1, err
	}
	defer func() { _ = body.Close() }()

	if err := xml.NewDecoder(body).Decode(target); err != nil {
		semp.logger.Error("Can't decode "+logName, "err", err, "broker", semp.brokerURI)
		return 0, err
	}

	result := target.SempReply().ExecuteResult
	if result.Result != "ok" {
		semp.logger.Error("unexpected result",
			"command", command,
			"result", result.Result,
			"reason", result.Reason,
			"broker", semp.brokerURI,
		)
		return 0, &unexpectedResultError{reason: result.Reason}
	}

	return 1, nil
}

// fetchSemp1Pages runs a paged SEMP v1 query: it fetches the first command, hands every successfully decoded page to
// handlePage and follows the broker's more-cookie until the last page. On a failing page it stops and returns that
// page's up value and error; pages handled before stay emitted.
func fetchSemp1Pages[T any, PT interface {
	*T
	semp1Reply
}](semp *Semp, command string, logName string, handlePage func(page PT)) (float64, error) {
	for page := 1; command != ""; page++ {
		target := PT(new(T))
		if up, err := semp.fetchSemp1(command, logName, page, target); err != nil {
			return up, err
		}
		command = target.SempReply().MoreCookie.RPC
		handlePage(target)
	}
	return 1, nil
}
