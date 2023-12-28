package rpc

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/pkg/errors"
)

const (
	rpcBlockPath        = "block"
	rpcBlockResultsPath = "block_results"
)

type Rpc interface {
	RemoteBlockHeight() (uint, error)
	Block(height uint) (*RpcRes[BlockRes], error)
	BlockResults(height uint) (*RpcRes[BlockResultRes], error)
}

type rpcImpl struct {
	baseUrl string
	client  *http.Client
}

func New(baseUrl string, client *http.Client) Rpc {
	return &rpcImpl{baseUrl, client}
}

// Block implements Rpc.
func (r *rpcImpl) Block(height uint) (*RpcRes[BlockRes], error) {
	url := fmt.Sprintf("%s/%s", r.baseUrl, rpcBlockPath)
	if height != 0 {
		url = fmt.Sprintf("%s?height=%d", url, height)
	}
	response, err := r.client.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "rpcImpl.Block")
	}
	defer response.Body.Close()

	data, _ := io.ReadAll(response.Body)

	var res RpcRes[BlockRes]
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, errors.Wrap(err, "rpcImpl.Block")
	}

	return &res, nil
}

// BlockResults implements Rpc.
func (r *rpcImpl) BlockResults(height uint) (*RpcRes[BlockResultRes], error) {
	url := fmt.Sprintf("%s/%s", r.baseUrl, rpcBlockResultsPath)
	if height != 0 {
		url = fmt.Sprintf("%s?height=%d", url, height)
	}
	response, err := r.client.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "rpcImpl.BlockResults")
	}
	defer response.Body.Close()

	data, _ := io.ReadAll(response.Body)

	var res RpcRes[BlockResultRes]
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, errors.Wrap(err, "rpcImpl.RemoteBlockHeight")
	}

	return &res, nil
}

// RemoteBlockHeight implements Rpc.
func (r *rpcImpl) RemoteBlockHeight() (uint, error) {
	response, err := r.client.Get(fmt.Sprintf("%s/%s", r.baseUrl, rpcBlockResultsPath))
	if err != nil {
		return 0, errors.Wrap(err, "rpcImpl.RemoteBlockHeight")
	}
	defer response.Body.Close()

	data, _ := io.ReadAll(response.Body)

	var res RpcRes[BlockResultRes]
	if err := json.Unmarshal(data, &res); err != nil {
		return 0, errors.Wrap(err, "rpcImpl.RemoteBlockHeight")
	}

	height, err := strconv.ParseInt(res.Result.Height, 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, "rpcImpl.RemoteBlockHeight")
	}
	return uint(height), nil
}
