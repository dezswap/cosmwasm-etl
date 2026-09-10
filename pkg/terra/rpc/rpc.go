package rpc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dezswap/cosmwasm-etl/pkg/httpclient"
	"github.com/dezswap/cosmwasm-etl/pkg/nodeerr"
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
	host    string
	client  *http.Client
}

func New(baseUrl string, client *http.Client) Rpc {
	if client.Timeout == 0 {
		cp := *client
		cp.Timeout = defaultRpcTimeout
		client = &cp
	}
	return &rpcImpl{baseUrl, nodeerr.HostOf(baseUrl), client}
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
		if !nodeerr.Retryable(err) {
			return nil, err
		}
		lastErr = err
	}

	return nil, fmt.Errorf("giving up after %d attempts: %w", maxAttempts, lastErr)
}

// get stamps the requested height on the failure. The message would otherwise not
// name it: the request URL that carries it is dropped to keep provider keys out.
func get[T any](r *rpcImpl, op, path string, height ...uint64) (*RpcRes[T], error) {
	res, err := do[T](r, op, path, height...)
	if err != nil && len(height) > 0 {
		err = nodeerr.WithHeight(err, height[0])
	}
	return res, err
}

func do[T any](r *rpcImpl, op, path string, height ...uint64) (*RpcRes[T], error) {
	url := fmt.Sprintf("%s/%s", r.baseUrl, path)
	if len(height) > 0 {
		url = fmt.Sprintf("%s?height=%d", url, height[0])
	}

	response, err := r.client.Get(url)
	if err != nil {
		return nil, nodeerr.Transient(op, nodeerr.TransportRPC, r.host, err)
	}
	defer response.Body.Close()

	// CometBFT answers a node error with HTTP 500 carrying the JSON-RPC object, so a
	// non-200 body still gets decoded. It is bounded because it is an error either
	// way: a JSON-RPC error object is orders of magnitude below the limit.
	var data []byte
	if response.StatusCode != http.StatusOK {
		data = httpclient.ReadErrorBody(response.Body)
	} else {
		data, err = httpclient.ReadBody(response.Body)
		if err != nil {
			return nil, &nodeerr.Error{Op: op, Transport: nodeerr.TransportRPC, Host: r.host, Status: response.StatusCode, Err: err, Class: nodeerr.ErrRetryable}
		}
	}

	var res RpcRes[T]
	if err := json.Unmarshal(data, &res); err != nil {
		if response.StatusCode != http.StatusOK {
			return nil, nodeerr.HTTPStatus(op, nodeerr.TransportRPC, r.host, response.StatusCode, data)
		}
		// An unparseable body behind HTTP 200 is a gateway artifact, not a node
		// verdict, so it is worth another attempt.
		return nil, &nodeerr.Error{
			Op: op, Transport: nodeerr.TransportRPC, Host: r.host,
			Status: response.StatusCode, Body: nodeerr.Snippet(data), Err: err, Class: nodeerr.ErrRetryable,
		}
	}

	if res.Error != nil {
		return nil, classifyRpcError(op, r.host, res.Error)
	}
	if response.StatusCode != http.StatusOK {
		return nil, nodeerr.HTTPStatus(op, nodeerr.TransportRPC, r.host, response.StatusCode, data)
	}

	return &res, nil
}
