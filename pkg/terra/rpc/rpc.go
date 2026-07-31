package rpc

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

const (
	rpcBlockPath        = "block"
	rpcBlockResultsPath = "block_results"
	rpcStatusPath       = "status"
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
	return &rpcImpl{baseUrl, client}
}

// Block implements Rpc.
func (r *rpcImpl) Block(height ...uint64) (*RpcRes[RpcBlockRes], error) {
	url := fmt.Sprintf("%s/%s", r.baseUrl, rpcBlockPath)
	if len(height) > 0 {
		url = fmt.Sprintf("%s?height=%d", url, height[0])
	}
	response, err := r.client.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "rpcImpl.Block")
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrap(err, "rpcImpl.Block")
	}

	var res RpcRes[RpcBlockRes]
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, errors.Wrap(err, "rpcImpl.Block")
	}

	return &res, nil
}

// BlockResults implements Rpc.
func (r *rpcImpl) BlockResults(height ...uint64) (*RpcRes[RpcBlockResultRes], error) {
	url := fmt.Sprintf("%s/%s", r.baseUrl, rpcBlockResultsPath)
	if len(height) > 0 {
		url = fmt.Sprintf("%s?height=%d", url, height[0])
	}
	response, err := r.client.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "rpcImpl.BlockResults")
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrap(err, "rpcImpl.BlockResults")
	}

	var res RpcRes[RpcBlockResultRes]
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, errors.Wrap(err, "rpcImpl.BlockResults")
	}

	return &res, nil
}

// Status implements Rpc.
func (r *rpcImpl) Status() (*RpcRes[RpcStatusRes], error) {
	url := fmt.Sprintf("%s/%s", r.baseUrl, rpcStatusPath)
	response, err := r.client.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "rpcImpl.Status")
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrap(err, "rpcImpl.Status")
	}

	var res RpcRes[RpcStatusRes]
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, errors.Wrap(err, "rpcImpl.Status")
	}

	return &res, nil
}
