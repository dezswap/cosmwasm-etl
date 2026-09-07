package rpc

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const maxBodySnippetLen = 256

// ErrHeightUnavailable marks a height the node can never serve; retrying cannot help.
var ErrHeightUnavailable = errors.New("height is not retained by the node")

// ErrRetryable marks a failure expected to clear on its own.
var ErrRetryable = errors.New("retryable node failure")

type NodeError struct {
	Op     string
	Status int
	Body   string
	Rpc    *RpcError
	Err    error
	// a nil class means permanent: errors.Is matches neither sentinel.
	class error
}

func (e *NodeError) Error() string {
	var msg string
	switch {
	case e.Rpc != nil:
		msg = fmt.Sprintf("%s: node returned error: %s", e.Op, e.Rpc)
	case e.Status != 0:
		msg = fmt.Sprintf("%s: node returned http %d", e.Op, e.Status)
		if e.Body != "" {
			msg += ": " + e.Body
		}
	default:
		msg = e.Op
	}
	if e.Err != nil {
		msg += ": " + e.Err.Error()
	}
	return msg
}

func (e *NodeError) Is(target error) bool { return target == e.class }

func (e *NodeError) Unwrap() error {
	if e.Rpc != nil {
		return e.Rpc
	}
	return e.Err
}

// CometBFT reports every failure as a generic -32603, so the message text is the
// only signal. "could not find results for height" is deliberately absent: it covers
// both a tip whose results are not written yet and a height whose ABCI responses were
// pruned, and only a caller that already fetched the block can tell those apart.
var permanentMarkers = []string{
	"is not available, lowest height is",              // rpc/core/env.go, below the block store base
	"node is not persisting finalize block responses", // state/store.go, discard_abci_responses
}

func classifyRpcError(op string, rpcErr *RpcError) error {
	class := error(ErrRetryable)
	for _, marker := range permanentMarkers {
		if strings.Contains(rpcErr.Data, marker) {
			class = ErrHeightUnavailable
			break
		}
	}
	return &NodeError{Op: op, Rpc: rpcErr, class: class}
}

func httpStatusError(op string, status int, body []byte) error {
	e := &NodeError{Op: op, Status: status, Body: bodySnippet(body)}
	if status >= http.StatusInternalServerError || status == http.StatusTooManyRequests {
		e.class = ErrRetryable
	}
	return e
}

func bodySnippet(body []byte) string {
	snippet := strings.TrimSpace(string(body))
	if len(snippet) > maxBodySnippetLen {
		return strings.ToValidUTF8(snippet[:maxBodySnippetLen], "") + "..."
	}
	return snippet
}
