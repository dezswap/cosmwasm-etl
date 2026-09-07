package rpc

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const prunedData = "height 100 is not available, lowest height is 200"

func newTestRpc(t *testing.T, handler http.HandlerFunc) Rpc {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return New(server.URL, server.Client())
}

// CometBFT's URI handler reports an RPC error as HTTP 500 carrying the error
// object, see rpc/jsonrpc/server/http_uri_handler.go.
func writeNodeError(w http.ResponseWriter, data string) {
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(w, `{"jsonrpc":"2.0","id":-1,"error":{"code":-32603,"message":"Internal error","data":%q}}`, data)
}

// The response carries no result, which used to decode as an empty success.
func TestBlockResultsRejectsNodeErrorResponse(t *testing.T) {
	client := newTestRpc(t, func(w http.ResponseWriter, _ *http.Request) {
		writeNodeError(w, prunedData)
	})

	res, err := client.BlockResults(100)

	require.Nil(t, res)
	require.ErrorContains(t, err, "rpcImpl.BlockResults: node returned error")
	require.ErrorContains(t, err, "lowest height is 200")

	var rpcErr *RpcError
	require.ErrorAs(t, err, &rpcErr)
	require.Equal(t, -32603, rpcErr.Code)
}

func TestBlockDoesNotRetryUnavailableHeight(t *testing.T) {
	var calls atomic.Int32
	client := newTestRpc(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		writeNodeError(w, prunedData)
	})

	_, err := client.Block(100)

	require.ErrorIs(t, err, ErrHeightUnavailable)
	require.Equal(t, int32(1), calls.Load())
}

func TestBlockResultsRetriesUntilNodeCatchesUp(t *testing.T) {
	var calls atomic.Int32
	client := newTestRpc(t, func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) < 3 {
			writeNodeError(w, "could not find results for height #100")
			return
		}
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":-1,"result":{"height":"100","txs_results":[]}}`)
	})

	res, err := client.BlockResults(100)

	require.NoError(t, err)
	require.Equal(t, "100", res.Result.Height)
	require.Equal(t, int32(3), calls.Load())
}

func TestBlockResultsGivesUpAfterMaxAttempts(t *testing.T) {
	var calls atomic.Int32
	client := newTestRpc(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := client.BlockResults(100)

	require.ErrorIs(t, err, ErrRetryable)
	require.Equal(t, int32(maxAttempts), calls.Load())
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

func TestStatusDoesNotRetryClientError(t *testing.T) {
	var calls atomic.Int32
	client := newTestRpc(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "no such endpoint")
	})

	_, err := client.Status()

	require.ErrorContains(t, err, "node returned http 404")
	require.False(t, errors.Is(err, ErrRetryable))
	require.Equal(t, int32(1), calls.Load())
}

func TestTransportFailureKeepsCause(t *testing.T) {
	client := New("http://127.0.0.1:1", &http.Client{Timeout: 50 * time.Millisecond})

	_, err := client.Status()

	require.ErrorIs(t, err, ErrRetryable)

	var urlErr *url.Error
	require.ErrorAs(t, err, &urlErr)
}

func TestClassifyRpcError(t *testing.T) {
	for _, tc := range []struct {
		name      string
		data      string
		permanent bool
	}{
		{"pruned block", "height 1 is not available, lowest height is 2", true},
		{"discarded abci responses", "node is not persisting finalize block responses", true},
		{"results not written yet", "could not find results for height #100", false},
		{"unknown failure", "something else went wrong", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := classifyRpcError("op", &RpcError{Data: tc.data})

			require.Equal(t, tc.permanent, errors.Is(err, ErrHeightUnavailable))
			require.Equal(t, !tc.permanent, errors.Is(err, ErrRetryable))
		})
	}
}
