package cosmos45

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dezswap/cosmwasm-etl/pkg/nodeerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The height header is absent on a non-200, so checking it before the status turned
// every gateway failure into a strconv error that classified as permanent.
func TestContractStateReportsTheStatusRatherThanTheMissingHeightHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("upstream connect error"))
	}))
	defer srv.Close()

	_, err := NewLcd(srv.URL, srv.Client()).ContractState("addr", `{"pool":{}}`, 12345)

	var nodeErr *nodeerr.Error
	require.ErrorAs(t, err, &nodeErr)
	assert.Equal(t, http.StatusBadGateway, nodeErr.Status)
	assert.Contains(t, nodeErr.Body, "upstream connect error")
	assert.True(t, nodeerr.Retryable(err))
	assert.NotContains(t, err.Error(), "ParseUint")
}

func TestContractStateStillRejectsAHeightTheNodeDidNotServe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Grpc-Metadata-X-Cosmos-Block-Height", "999")
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	defer srv.Close()

	_, err := NewLcd(srv.URL, srv.Client()).ContractState("addr", `{"pool":{}}`, 12345)

	require.ErrorContains(t, err, "invalid height, expected 12345, got 999")
}

func TestContractStateReturnsThePayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Grpc-Metadata-X-Cosmos-Block-Height", "12345")
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	defer srv.Close()

	data, err := NewLcd(srv.URL, srv.Client()).ContractState("addr", `{"pool":{}}`, 12345)

	require.NoError(t, err)
	assert.Equal(t, `{"data":{}}`, string(data))
}
