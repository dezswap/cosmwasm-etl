package datastore

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	tm_types "github.com/cometbft/cometbft/types"
	cosmos_types "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"

	// Aliased: the local httpClient interface below differs only by case.
	httpx "github.com/dezswap/cosmwasm-etl/pkg/httpclient"
	"github.com/dezswap/cosmwasm-etl/pkg/nodeerr"
	"github.com/pkg/errors"
)

type httpClient interface {
	Get(url string) (*http.Response, error)
}

type LcdClient interface {
	GetTx(txHash string) (*txtypes.GetTxResponse, error)
	GetBlockWithTxs(height int64) (*txtypes.GetBlockWithTxsResponse, error)
}

const (
	lcdTxQueryPath    = "cosmos/tx/v1beta1/txs"
	lcdBlockQueryPath = "blocks"
)

type lcdClientImpl struct {
	baseUrl string
	host    string
	httpClient
}

var _ LcdClient = &lcdClientImpl{}

func NewLcdClient(baseUrl string, c httpClient) LcdClient {
	return &lcdClientImpl{baseUrl, nodeerr.HostOf(baseUrl), c}
}

func (c *lcdClientImpl) read(op, url string) ([]byte, error) {
	response, err := c.Get(url)
	if err != nil {
		return nil, nodeerr.Transient(op, nodeerr.TransportLCD, c.host, err)
	}
	defer response.Body.Close()

	return httpx.ReadResponse(op, nodeerr.TransportLCD, c.host, response)
}

// GetTx only returns TxResponse
func (c *lcdClientImpl) GetTx(txHash string) (*txtypes.GetTxResponse, error) {
	const op = "lcdClientImpl.GetTx"

	data, err := c.read(op, fmt.Sprintf("%s/%s/%s", c.baseUrl, lcdTxQueryPath, txHash))
	if err != nil {
		return nil, err
	}

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

	var res txRes
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, unexpectedLcdBody(op, c.host, data, err)
	}

	// A 200 whose body carries no tx_response is a shape the node was not supposed
	// to return, and the embedded pointer stays nil, so report it before dereferencing.
	if res.OverriddenRes.TxResponse == nil {
		return nil, unexpectedLcdBody(op, c.host, data, errors.New("response has no tx_response"))
	}

	height, err := strconv.ParseInt(res.OverriddenRes.Height, 10, 64)
	if err != nil {
		return nil, unexpectedLcdBody(op, c.host, data, err)
	}
	res.OverriddenRes.TxResponse.Height = height

	gasWanted, err := strconv.ParseInt(res.OverriddenRes.GasWanted, 10, 64)
	if err != nil {
		return nil, unexpectedLcdBody(op, c.host, data, err)
	}
	res.OverriddenRes.TxResponse.GasWanted = gasWanted

	gasUsed, err := strconv.ParseInt(res.OverriddenRes.GasUsed, 10, 64)
	if err != nil {
		return nil, unexpectedLcdBody(op, c.host, data, err)
	}
	res.OverriddenRes.TxResponse.GasUsed = gasUsed
	res.OverriddenRes.Tx = nil

	return &txtypes.GetTxResponse{
		Tx:         nil,
		TxResponse: res.OverriddenRes.TxResponse,
	}, nil
}

// GetBlockWithTxs implements lcdClient.
func (c *lcdClientImpl) GetBlockWithTxs(height int64) (*txtypes.GetBlockWithTxsResponse, error) {
	const op = "lcdClientImpl.GetBlockWithTxs"

	data, err := c.read(op, fmt.Sprintf("%s/%s/%d", c.baseUrl, lcdBlockQueryPath, height))
	if err != nil {
		return nil, err
	}

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

	var res blockRes
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, unexpectedLcdBody(op, c.host, data, err)
	}

	return &txtypes.GetBlockWithTxsResponse{}, nil
}

// unexpectedLcdBody reports a body the node answered with HTTP 200 but that this
// client cannot use. It is retryable because a well formed 200 that does not decode
// is usually a proxy artifact rather than the node's own answer.
func unexpectedLcdBody(op, host string, body []byte, cause error) error {
	return &nodeerr.Error{
		Op: op, Transport: nodeerr.TransportLCD, Host: host,
		Status: http.StatusOK, Body: nodeerr.Snippet(body), Err: cause, Class: nodeerr.ErrRetryable,
	}
}
