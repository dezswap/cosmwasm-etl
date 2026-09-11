package httpclient

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/dezswap/cosmwasm-etl/pkg/nodeerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadBodyReturnsTheWholePayload(t *testing.T) {
	data, err := ReadBody(strings.NewReader(`{"height":"1"}`))

	require.NoError(t, err)
	require.Equal(t, `{"height":"1"}`, string(data))
}

// Truncating instead would hand the caller a payload that fails to decode far from
// where the limit was hit, so the limit has to surface as its own error.
func TestReadBodyFailsInsteadOfTruncating(t *testing.T) {
	_, err := ReadBody(io.LimitReader(neverEnding{}, MaxBodyBytes+1))

	require.ErrorContains(t, err, "exceeds")
}

func TestReadBodyAcceptsExactlyTheLimit(t *testing.T) {
	data, err := ReadBody(io.LimitReader(neverEnding{}, MaxBodyBytes))

	require.NoError(t, err)
	require.Len(t, data, MaxBodyBytes)
}

func TestReadErrorBodyTruncatesAtTheLimit(t *testing.T) {
	body := ReadErrorBody(strings.NewReader(strings.Repeat("x", MaxErrorBodyBytes*2)))

	assert.Len(t, body, MaxErrorBodyBytes)
}

// The status code is the more useful half of the report, so a body that cannot be
// read must not become an error that replaces it.
func TestReadErrorBodyNeverFails(t *testing.T) {
	body := ReadErrorBody(iotest.ErrReader(errors.New("connection reset")))

	assert.Empty(t, body)
}

func TestReadErrorBodyKeepsAShortBodyWhole(t *testing.T) {
	assert.Equal(t, "upstream connect error", string(ReadErrorBody(strings.NewReader("upstream connect error"))))
}

func TestReadResponseReturnsThePayloadOn200(t *testing.T) {
	res := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}

	data, err := ReadResponse("op", nodeerr.TransportLCD, "node:1317", res)

	require.NoError(t, err)
	assert.Equal(t, `{"ok":true}`, string(data))
}

// A gateway error page decodes into a zero value that reads like an empty result,
// so the status has to stop it before it ever reaches a decoder.
func TestReadResponseRejectsANonOKBodyWithItsStatus(t *testing.T) {
	res := &http.Response{
		StatusCode: http.StatusBadGateway,
		Body:       io.NopCloser(strings.NewReader("<html>upstream connect error</html>")),
	}

	_, err := ReadResponse("op", nodeerr.TransportLCD, "node:1317", res)

	var nodeErr *nodeerr.Error
	require.ErrorAs(t, err, &nodeErr)
	assert.Equal(t, http.StatusBadGateway, nodeErr.Status)
	assert.Contains(t, nodeErr.Body, "upstream connect error")
	assert.True(t, nodeerr.Retryable(err))
}

func TestReadResponseRejectsAnOversizedPayload(t *testing.T) {
	res := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(io.LimitReader(neverEnding{}, MaxBodyBytes+1)),
	}

	_, err := ReadResponse("op", nodeerr.TransportRPC, "node:26657", res)

	require.Error(t, err)
	assert.True(t, nodeerr.Retryable(err))
}

type neverEnding struct{}

func (neverEnding) Read(p []byte) (int, error) { return len(p), nil }
