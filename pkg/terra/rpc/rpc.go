package rpc

import (
	"encoding/json"
	"fmt"
	"github.com/dezswap/cosmwasm-etl/pkg/util"
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
	RemoteBlockHeight() (uint64, error)
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

	data, _ := io.ReadAll(response.Body)

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

	data, _ := io.ReadAll(response.Body)

	var res RpcRes[RpcBlockResultRes]
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, errors.Wrap(err, "rpcImpl.BlockResults")
	}

	return &res, nil
}

// RemoteBlockHeight implements Rpc.
func (r *rpcImpl) RemoteBlockHeight() (uint64, error) {
	response, err := r.client.Get(fmt.Sprintf("%s/%s", r.baseUrl, rpcBlockResultsPath))
	if err != nil {
		return 0, errors.Wrap(err, "rpcImpl.RemoteBlockHeight")
	}
	defer response.Body.Close()

	data, _ := io.ReadAll(response.Body)

	var res RpcRes[RpcBlockResultRes]
	if err := json.Unmarshal(data, &res); err != nil {
		return 0, errors.Wrapf(
			err, "rpcImpl.RemoteBlockHeight: failed to unmarshal data, first %d bytes: %s",
			util.DefaultErrorDataLength, util.TruncateBytes(data, util.DefaultErrorDataLength))
	}

	height, err := strconv.ParseInt(res.Result.Height, 10, 64)
	if err != nil {
		return 0, errors.Wrapf(
			err, "rpcImpl.RemoteBlockHeight: failed to parse int, first %d bytes: %s",
			util.DefaultErrorDataLength, util.TruncateBytes(data, util.DefaultErrorDataLength))
	}
	return uint64(height), nil
}
