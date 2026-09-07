package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	rpcBlockPath        = "block"
	rpcBlockResultsPath = "block_results"
	rpcStatusPath       = "status"

	defaultRpcTimeout = 30 * time.Second

	maxAttempts    = 3
	initialBackoff = 200 * time.Millisecond
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
	return getWithRetry[RpcBlockRes](r, "rpcImpl.Block", rpcBlockPath, height...)
}

// BlockResults implements Rpc.
func (r *rpcImpl) BlockResults(height ...uint64) (*RpcRes[RpcBlockResultRes], error) {
	return getWithRetry[RpcBlockResultRes](r, "rpcImpl.BlockResults", rpcBlockResultsPath, height...)
}

// Status implements Rpc.
func (r *rpcImpl) Status() (*RpcRes[RpcStatusRes], error) {
	return getWithRetry[RpcStatusRes](r, "rpcImpl.Status", rpcStatusPath)
}

// getWithRetry retries only failures classified as retryable. A height the node
// does not retain fails immediately so the caller can quarantine it instead of
// spinning on a request that can never succeed.
func getWithRetry[T any](r *rpcImpl, op, path string, height ...uint64) (*RpcRes[T], error) {
	backoff := initialBackoff
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(backoff)
			backoff *= 2
		}

		res, err := get[T](r, op, path, height...)
		if err == nil {
			return res, nil
		}
		if !errors.Is(err, ErrRetryable) {
			return nil, err
		}
		lastErr = err
	}

	return nil, fmt.Errorf("giving up after %d attempts: %w", maxAttempts, lastErr)
}

func get[T any](r *rpcImpl, op, path string, height ...uint64) (*RpcRes[T], error) {
	url := fmt.Sprintf("%s/%s", r.baseUrl, path)
	if len(height) > 0 {
		url = fmt.Sprintf("%s?height=%d", url, height[0])
	}

	response, err := r.client.Get(url)
	if err != nil {
		// Transport failures carry no node verdict, so they stay retryable.
		return nil, &NodeError{Op: op, Err: err, class: ErrRetryable}
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, &NodeError{Op: op, Status: response.StatusCode, Err: err, class: ErrRetryable}
	}

	var res RpcRes[T]
	if err := json.Unmarshal(data, &res); err != nil {
		if response.StatusCode != http.StatusOK {
			return nil, httpStatusError(op, response.StatusCode, data)
		}
		// An unparseable body behind HTTP 200 is a gateway artifact, not a node
		// verdict, so it is worth another attempt.
		return nil, &NodeError{Op: op, Status: response.StatusCode, Body: bodySnippet(data), Err: err, class: ErrRetryable}
	}

	if res.Error != nil {
		return nil, classifyRpcError(op, res.Error)
	}
	if response.StatusCode != http.StatusOK {
		return nil, httpStatusError(op, response.StatusCode, data)
	}

	return &res, nil
}
