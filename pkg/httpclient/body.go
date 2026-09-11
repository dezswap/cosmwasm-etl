package httpclient

import (
	"fmt"
	"io"
	"net/http"

	"github.com/dezswap/cosmwasm-etl/pkg/nodeerr"
)

const (
	// A block with all its txs, the largest payload asked for here, stays far below this.
	MaxBodyBytes = 64 << 20
	// Enough for a gateway error page to survive into a nodeerr.Snippet.
	MaxErrorBodyBytes = 32 << 10
)

// ReadResponse returns the body of a response the caller can decode, and a
// *nodeerr.Error for anything else.
func ReadResponse(op, transport, host string, res *http.Response) ([]byte, error) {
	if res.StatusCode != http.StatusOK {
		return nil, nodeerr.HTTPStatus(op, transport, host, res.StatusCode, ReadErrorBody(res.Body))
	}

	data, err := ReadBody(res.Body)
	if err != nil {
		return nil, &nodeerr.Error{
			Op: op, Transport: transport, Host: host,
			Status: res.StatusCode, Err: err, Class: nodeerr.ErrRetryable,
		}
	}

	return data, nil
}

// ReadBody buffers a body the caller intends to decode, failing past MaxBodyBytes
// rather than truncating: a silently cut payload would surface as an unexplained
// JSON syntax error somewhere far from here.
func ReadBody(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxBodyBytes {
		return nil, fmt.Errorf("response body exceeds %d bytes", MaxBodyBytes)
	}
	return data, nil
}

// ReadErrorBody buffers as much of a body as a nodeerr.Snippet can use.
func ReadErrorBody(r io.Reader) []byte {
	body, _ := io.ReadAll(io.LimitReader(r, MaxErrorBodyBytes))
	return body
}
