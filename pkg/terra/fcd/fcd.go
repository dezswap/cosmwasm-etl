package fcd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

const (
	FCD_TXS_MAX_OFFSET = 0x7FFFFFFF
	FCD_TXS_MIN_LIMIT  = 10
	FCD_TXS_MAX_LIMIT  = 100
)

var (
	STATUS_SERVER_ERROR = errors.New("status code is bigger than or equal 500")
)

type Fcd interface {
	TxsOf(addr string, option FcdTxsReqQuery) (*FcdTxsRes, error)
}

type fcdImpl struct {
	baseUrl string
	*http.Client
}

// TxsOf implements Fcd.
func (f *fcdImpl) TxsOf(addr string, option FcdTxsReqQuery) (*FcdTxsRes, error) {
	next, limit := 0, FCD_TXS_MAX_LIMIT
	if option.Offset != nil {
		next = *option.Offset
	}
	if option.Limit != nil {
		limit = *option.Limit
		if limit < FCD_TXS_MIN_LIMIT {
			limit = FCD_TXS_MIN_LIMIT
		}
	}

	params := url.Values{}
	params.Add("account", addr)
	params.Add("offset", fmt.Sprintf("%d", next))
	params.Add("limit", fmt.Sprintf("%d", limit))
	reqUrl := fmt.Sprintf("%s/v1/txs?%s", f.baseUrl, params.Encode())

	res, err := f.Client.Get(reqUrl)
	if err != nil {
		return nil, errors.Wrap(err, "fcdImpl.TxsOf")
	}

	defer res.Body.Close()

	if res.StatusCode >= http.StatusInternalServerError {
		return nil, errors.Wrapf(STATUS_SERVER_ERROR, "fcdImpl.TxsOf: status code %d", res.StatusCode)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "fcdImpl.TxsOf")
	}

	txsRes := FcdTxsRes{}
	if err := json.Unmarshal(data, &txsRes); err != nil {
		return nil, errors.Wrap(err, "fcdImpl.TxsOf")
	}

	return &txsRes, nil
}

func New(baseUrl string, client *http.Client) Fcd {
	return &fcdImpl{baseUrl, client}
}

type FcdTxsReqQuery struct {
	Limit  *int `json:"limit"`
	Offset *int `json:"next"`
}

type FcdTxsRes struct {
	Limit int        `json:"limit"`
	Next  int        `json:"next"`
	Txs   []FcdTxRes `json:"txs"`
}

type FcdTxRes struct {
	Id        int           `json:"id"`
	ChainId   string        `json:"chainId"`
	Logs      []FcdTxLogRes `json:"logs"`
	Height    string        `json:"height"`
	TxHash    string        `json:"txhash"`
	RawLog    string        `json:"raw_log"`
	Timestamp string        `json:"timestamp"`
}

type FcdTxLogRes struct {
	MsgIndex int                `json:"msg_index"`
	Events   []FcdTxLogEventRes `json:"events"`
}

type FcdTxLogEventRes struct {
	Type       string `json:"type"`
	Attributes []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"attributes"`
}
