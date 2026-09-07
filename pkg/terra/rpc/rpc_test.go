package rpc

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestRpc(t *testing.T, handler http.HandlerFunc) Rpc {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return New(server.URL, server.Client())
}

// A JSON-RPC error arrives with HTTP 200 and no result, which used to decode as
// an empty successful response and surfaced later as a txs length mismatch.
func TestBlockResultsRejectsNodeErrorResponse(t *testing.T) {
	client := newTestRpc(t, func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":-1,"error":{"code":-32603,"message":"Internal error","data":"height 100 is not available, lowest height is 200"}}`)
	})

	res, err := client.BlockResults(100)

	require.Nil(t, res)
	require.ErrorContains(t, err, "rpcImpl.BlockResults: node returned error")
	require.ErrorContains(t, err, "lowest height is 200")

	var rpcErr *RpcError
	require.ErrorAs(t, err, &rpcErr)
	require.Equal(t, -32603, rpcErr.Code)
}

func TestBlockRejectsNodeErrorResponse(t *testing.T) {
	client := newTestRpc(t, func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":-1,"error":{"code":-32603,"message":"Internal error","data":"could not find results for height #100"}}`)
	})

	_, err := client.Block(100)

	require.ErrorContains(t, err, "could not find results for height #100")
}

func TestStatusRejectsNonOkStatusCode(t *testing.T) {
	client := newTestRpc(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, "rate limited")
	})

	_, err := client.Status()

	require.ErrorContains(t, err, "node returned http 429")
	require.ErrorContains(t, err, "rate limited")
}

func TestBlockResultsReturnsResultOnSuccess(t *testing.T) {
	client := newTestRpc(t, func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":-1,"result":{"height":"100","txs_results":[]}}`)
	})

	res, err := client.BlockResults(100)

	require.NoError(t, err)
	require.Equal(t, "100", res.Result.Height)
}
