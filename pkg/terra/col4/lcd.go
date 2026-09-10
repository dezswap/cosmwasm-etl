package col4

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/dezswap/cosmwasm-etl/pkg/httpclient"
	"github.com/dezswap/cosmwasm-etl/pkg/nodeerr"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/lcd"
	"github.com/pkg/errors"
)

type lcdImpl struct {
	baseUrl string
	host    string
	client  *http.Client
}

func NewLcd(baseUrl string, client *http.Client) lcd.Lcd[LcdTxRes] {
	return &lcdImpl{baseUrl, nodeerr.HostOf(baseUrl), client}
}

// Tx implements Lcd.
func (l *lcdImpl) Tx(hash string) (*LcdTxRes, error) {
	const op = "lcdImpl.Tx"

	reqUrl := fmt.Sprintf("%s/txs/%s", l.baseUrl, hash)
	response, err := l.client.Get(reqUrl)
	if err != nil {
		return nil, nodeerr.Transient(op, nodeerr.TransportLCD, l.host, err)
	}
	defer response.Body.Close()

	data, err := httpclient.ReadResponse(op, nodeerr.TransportLCD, l.host, response)
	if err != nil {
		return nil, err
	}
	LcdTxRes := LcdTxRes{}

	if err := json.Unmarshal(data, &LcdTxRes); err != nil {
		return nil, errors.Wrap(err, "lcdImpl.Tx")
	}

	return &LcdTxRes, nil
}

func (l *lcdImpl) ContractState(address string, query string, height ...uint64) ([]byte, error) {
	const op = "lcdImpl.ContractState"

	params := url.Values{}
	params.Add("query_msg", query)
	if len(height) > 0 {
		params.Add("height", fmt.Sprintf("%d", height[0]))
	}

	reqUrl := fmt.Sprintf("%s/wasm/contracts/%s/store?%s", l.baseUrl, address, params.Encode())
	response, err := l.client.Get(reqUrl)
	if err != nil {
		return nil, nodeerr.Transient(op, nodeerr.TransportLCD, l.host, err)
	}
	defer response.Body.Close()

	return httpclient.ReadResponse(op, nodeerr.TransportLCD, l.host, response)
}

func QueryContractState[T any](lcd lcd.Lcd[LcdTxRes], address string, query string, height ...uint64) (*LcdContractStateRes[T], error) {
	resBytes, err := lcd.ContractState(address, query, height...)
	if err != nil {
		return nil, err
	}

	var result LcdContractStateRes[T]
	if err := json.Unmarshal(resBytes, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
