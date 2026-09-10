package fcd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A 4xx body decodes into a zero value, which the caller would read as an account
// with no txs rather than as a failed request.
func TestTxsOfRejectsAClientError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"account not found"}`))
	}))
	defer srv.Close()

	_, err := New(srv.URL, srv.Client()).TxsOf("terra1abc", FcdTxsReqQuery{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
	assert.Contains(t, err.Error(), "account not found")
}

// The retry loop in collector/terra/fcd matches on this sentinel's text.
func TestTxsOfKeepsTheServerErrorSentinel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	_, err := New(srv.URL, srv.Client()).TxsOf("terra1abc", FcdTxsReqQuery{})

	require.ErrorIs(t, err, STATUS_SERVER_ERROR)
	assert.Contains(t, err.Error(), STATUS_SERVER_ERROR.Error())
}

func TestTxsOfReturnsTheDecodedPayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"next":42,"txs":[]}`))
	}))
	defer srv.Close()

	res, err := New(srv.URL, srv.Client()).TxsOf("terra1abc", FcdTxsReqQuery{})

	require.NoError(t, err)
	assert.Equal(t, 42, res.Next)
}
