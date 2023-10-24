package datastore

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	cosmos_types "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/pkg/errors"
	tm_types "github.com/tendermint/tendermint/types"
)

type httpClient interface {
	Get(url string) (*http.Response, error)
}

type lcdClient interface {
	GetTx(txHash string) (*txtypes.GetTxResponse, error)
	GetBlockWithTxs(height int64) (*txtypes.GetBlockWithTxsResponse, error)
}

const (
	lcdTxQueryPath    = "cosmos/tx/v1beta1/txs"
	lcdBlockQueryPath = "blocks"
)

type lcdClientImpl struct {
	baseUrl string
	httpClient
}

var _ lcdClient = &lcdClientImpl{}

func NewLcdClient(baseUrl string, c httpClient) lcdClient {
	return &lcdClientImpl{baseUrl, c}
}

// GetTx only returns TxResponse
func (c *lcdClientImpl) GetTx(txHash string) (*txtypes.GetTxResponse, error) {

	response, err := http.Get(fmt.Sprintf("%s/%s/%s", c.baseUrl, lcdTxQueryPath, txHash))
	if err != nil {
		return nil, errors.Wrap(err, "lcdClientImpl.GetTx")
	}
	defer response.Body.Close()

	type txRes struct {
		Tx interface{} `json:"tx,omitempty"`
		// tx_response is the queried TxResponses.
		OverriddenRes struct {
			*cosmos_types.TxResponse
			Height    string `json:"height"`
			GasWanted string `json:"gas_wanted"`
			GasUsed   string `json:"gas_used"`
		} `protobuf:"bytes,2,opt,name=tx_response,json=txResponse,proto3" json:"tx_response,omitempty"`
	}

	data, _ := io.ReadAll(response.Body)

	var res txRes
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, errors.Wrap(err, "lcdClientImpl.GetTx")
	}

	height, err := strconv.ParseInt(res.OverriddenRes.Height, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "lcdClientImpl.GetTx")
	}
	res.OverriddenRes.TxResponse.Height = height

	gasWanted, err := strconv.ParseInt(res.OverriddenRes.GasWanted, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "lcdClientImpl.GetTx")
	}
	res.OverriddenRes.TxResponse.GasWanted = gasWanted

	gasUsed, err := strconv.ParseInt(res.OverriddenRes.GasUsed, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "lcdClientImpl.GetTx")
	}
	res.OverriddenRes.TxResponse.GasUsed = gasUsed
	res.OverriddenRes.TxResponse.Tx = nil

	return &txtypes.GetTxResponse{
		Tx:         nil,
		TxResponse: res.OverriddenRes.TxResponse,
	}, nil
}

// GetBlockWithTxs implements lcdClient.
func (c *lcdClientImpl) GetBlockWithTxs(height int64) (*txtypes.GetBlockWithTxsResponse, error) {

	response, err := http.Get(fmt.Sprintf("%s/%s/%d", c.baseUrl, lcdBlockQueryPath, height))
	if err != nil {
		return nil, errors.Wrap(err, "lcdClientImpl.GetBlockWithTxs")
	}
	defer response.Body.Close()

	type headerRes struct {
		tm_types.Header
		Version interface{} `json:"version"`
		Height  string      `json:"height"`
	}

	type commitRes struct {
		tm_types.Commit
		Height string `json:"height"`
	}

	type blockRes struct {
		BlockId tm_types.BlockID `json:"block_id,omitempty"`
		Block   struct {
			tm_types.Block
			Header     headerRes `json:"header,omitempty"`
			LastCommit commitRes `json:"last_commit,omitempty"`
		} `json:"block,omitempty"`
	}

	data, _ := io.ReadAll(response.Body)
	var res blockRes
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, errors.Wrap(err, "lcdClientImpl.GetBlockWithTxs")
	}

	return &txtypes.GetBlockWithTxsResponse{}, nil
}
