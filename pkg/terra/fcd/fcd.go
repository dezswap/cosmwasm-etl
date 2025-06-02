package fcd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

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
	Block(height uint64, chainId string) (*FcdBlockRes, error)
	TxsOf(addr string, option FcdTxsReqQuery) (*FcdTxsRes, error)
	Tx(hash string) (*FcdTxRes, error)
}

type fcdImpl struct {
	baseUrl string
	*http.Client
}

func (f *fcdImpl) Block(height uint64, chainId string) (*FcdBlockRes, error) {
	u, err := url.Parse(f.baseUrl)
	u = u.JoinPath("v1", "blocks", strconv.Itoa(int(height)))
	params := url.Values{}
	params.Add("chainId", chainId)
	u.RawQuery = params.Encode()

	res, err := f.Client.Get(u.String())
	if err != nil {
		return nil, errors.Wrap(err, "fcdImpl.Block")
	}

	defer res.Body.Close()
	if res.StatusCode >= http.StatusInternalServerError {
		return nil, errors.Wrapf(STATUS_SERVER_ERROR, "fcdImpl.Block: status code %d", res.StatusCode)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "fcdImpl.Block")
	}

	blockRes := FcdBlockRes{}
	if err := json.Unmarshal(data, &blockRes); err != nil {
		return nil, errors.Wrap(err, "fcdImpl.Tx")
	}

	return &blockRes, nil
}

func (f *fcdImpl) Tx(hash string) (*FcdTxRes, error) {
	u, err := url.Parse(f.baseUrl)
	u = u.JoinPath("v1", "tx", hash)

	res, err := f.Client.Get(u.String())
	if err != nil {
		return nil, errors.Wrap(err, "fcdImpl.Tx")
	}

	defer res.Body.Close()
	if res.StatusCode >= http.StatusInternalServerError {
		return nil, errors.Wrapf(STATUS_SERVER_ERROR, "fcdImpl.Tx: status code %d", res.StatusCode)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "fcdImpl.Tx")
	}

	txRes := FcdTxRes{}
	if err := json.Unmarshal(data, &txRes); err != nil {
		return nil, errors.Wrap(err, "fcdImpl.Tx")
	}

	return &txRes, nil
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
		if limit > FCD_TXS_MAX_LIMIT {
			limit = FCD_TXS_MAX_LIMIT
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
