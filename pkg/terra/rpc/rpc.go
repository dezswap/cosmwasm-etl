package rpc

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	rpcBlockPath        = "block"
	rpcBlockResultsPath = "block_results"
	rpcStatusPath       = "status"

	defaultRpcTimeout = 30 * time.Second

	// node error pages are unbounded, so only an excerpt reaches the log
	maxBodySnippetLen = 256
)

type Rpc interface {
	Status() (*RpcRes[RpcStatusRes], error)
	Block(height ...uint64) (*RpcRes[RpcBlockRes], error)
	BlockResults(height ...uint64) (*RpcRes[RpcBlockResultRes], error)
}

type rpcImpl struct {
	baseUrl string
	client  *http.Client
}

func New(baseUrl string, client *http.Client) Rpc {
	if client.Timeout == 0 {
		client.Timeout = defaultRpcTimeout
	}
	return &rpcImpl{baseUrl, client}
}

// Block implements Rpc.
func (r *rpcImpl) Block(height ...uint64) (*RpcRes[RpcBlockRes], error) {
	return get[RpcBlockRes](r, "rpcImpl.Block", rpcBlockPath, height...)
}

// BlockResults implements Rpc.
func (r *rpcImpl) BlockResults(height ...uint64) (*RpcRes[RpcBlockResultRes], error) {
	return get[RpcBlockResultRes](r, "rpcImpl.BlockResults", rpcBlockResultsPath, height...)
}

// Status implements Rpc.
func (r *rpcImpl) Status() (*RpcRes[RpcStatusRes], error) {
	return get[RpcStatusRes](r, "rpcImpl.Status", rpcStatusPath)
}

// get fails on a node error instead of decoding it as an empty success. CometBFT
// reports errors with HTTP 200 and no result field, so a caller that only reads
// Result cannot tell a pruned or not yet available height from an empty block.
func get[T any](r *rpcImpl, op, path string, height ...uint64) (*RpcRes[T], error) {
	url := fmt.Sprintf("%s/%s", r.baseUrl, path)
	if len(height) > 0 {
		url = fmt.Sprintf("%s?height=%d", url, height[0])
	}

	response, err := r.client.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, op)
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrap(err, op)
	}

	var res RpcRes[T]
	if err := json.Unmarshal(data, &res); err != nil {
		if response.StatusCode != http.StatusOK {
			return nil, httpStatusError(op, response.StatusCode, data)
		}
		return nil, errors.Wrap(err, op)
	}

	// Wrapped rather than formatted so callers can errors.As the RpcError and
	// tell a permanently pruned height from a transient node failure.
	if res.Error != nil {
		return nil, errors.Wrapf(res.Error, "%s: node returned error (code %d)", op, res.Error.Code)
	}
	if response.StatusCode != http.StatusOK {
		return nil, httpStatusError(op, response.StatusCode, data)
	}

	return &res, nil
}

func httpStatusError(op string, statusCode int, body []byte) error {
	if snippet := bodySnippet(body); snippet != "" {
		return errors.Errorf("%s: node returned http %d: %s", op, statusCode, snippet)
	}
	return errors.Errorf("%s: node returned http %d", op, statusCode)
}

func bodySnippet(body []byte) string {
	snippet := strings.TrimSpace(string(body))
	if len(snippet) > maxBodySnippetLen {
		return strings.ToValidUTF8(snippet[:maxBodySnippetLen], "") + "..."
	}
	return snippet
}
