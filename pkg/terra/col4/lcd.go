package col4

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

type Lcd interface {
	ContractState(address string, query string, height ...uint64) ([]byte, error)
}

type lcdImpl struct {
	baseUrl string
	client  *http.Client
}

func NewLcd(baseUrl string, client *http.Client) Lcd {
	return &lcdImpl{baseUrl, client}
}

func (l *lcdImpl) ContractState(address string, query string, height ...uint64) ([]byte, error) {
	params := url.Values{}
	params.Add("query_msg", query)
	if len(height) > 0 {
		params.Add("height", fmt.Sprintf("%d", height[0]))
	}

	reqUrl := fmt.Sprintf("%s/wasm/contracts/%s/store?%s", l.baseUrl, address, params.Encode())
	response, err := l.client.Get(reqUrl)
	if err != nil {
		return nil, errors.Wrap(err, "lcdImpl.ContractStateRes")
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrap(err, "lcdImpl.ContractStateRes")
	}

	return data, nil
}

func QueryContractState[T any](lcd Lcd, address string, query string, height ...uint64) (*LcdContractStateRes[T], error) {
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
