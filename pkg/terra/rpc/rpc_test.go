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

const prunedBody = `{"jsonrpc":"2.0","id":-1,"error":{"code":-32603,"message":"Internal error","data":"height 100 is not available, lowest height is 200"}}`

func newTestRpc(t *testing.T, handler http.HandlerFunc) Rpc {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return New(server.URL, server.Client())
}

// A JSON-RPC error arrives with HTTP 200 and no result, which used to decode as
// an empty successful response.
func TestBlockResultsRejectsNodeErrorResponse(t *testing.T) {
	client := newTestRpc(t, func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, prunedBody)
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
		fmt.Fprint(w, prunedBody)
	})

	_, err := client.Block(100)

	require.ErrorIs(t, err, ErrHeightUnavailable)
	require.Equal(t, int32(1), calls.Load())
}

func TestBlockResultsRetriesUntilNodeCatchesUp(t *testing.T) {
	var calls atomic.Int32
	client := newTestRpc(t, func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) < 3 {
			fmt.Fprint(w, `{"jsonrpc":"2.0","id":-1,"error":{"code":-32603,"message":"Internal error","data":"could not find results for height #100"}}`)
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
