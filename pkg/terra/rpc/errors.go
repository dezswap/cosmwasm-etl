package rpc

import (
	"strings"

	"github.com/dezswap/cosmwasm-etl/pkg/nodeerr"
)

// CometBFT reports every failure as a generic -32603, so the message text is the
// only signal. "could not find results for height" is deliberately absent: it covers
// both a tip whose results are not written yet and a height whose ABCI responses were
// pruned, and only a caller that already fetched the block can tell those apart.
var permanentMarkers = []string{
	"is not available, lowest height is",              // rpc/core/env.go, below the block store base
	"node is not persisting finalize block responses", // state/store.go, discard_abci_responses
}

func classifyRpcError(op, host string, rpcErr *RpcError) error {
	class := error(nodeerr.ErrRetryable)
	for _, marker := range permanentMarkers {
		if strings.Contains(rpcErr.Data, marker) {
			class = nodeerr.ErrHeightUnavailable
			break
		}
	}
	// Code is left unset so the message renders the JSON-RPC text: CometBFT reports
	// everything as -32603, which on its own tells a reader nothing.
	return &nodeerr.Error{
		Op:        op,
		Transport: nodeerr.TransportRPC,
		Host:      host,
		Detail:    rpcErr,
		Class:     class,
	}
}
