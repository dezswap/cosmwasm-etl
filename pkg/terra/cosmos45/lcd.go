package cosmos45

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/dezswap/cosmwasm-etl/pkg/terra"
	"github.com/pkg/errors"
)

type Lcd interface {
	ContractState(address string, query string, height ...uint64) ([]byte, error)
	Tx(hash string) (*LcdTxRes, error)
}

type lcdImpl struct {
	baseUrl string
	client  *http.Client
}

func NewLcd(baseUrl string, client *http.Client) Lcd {
	return &lcdImpl{baseUrl, client}
}

// Tx implements Lcd.
func (l *lcdImpl) Tx(hash string) (*LcdTxRes, error) {
	reqUrl := fmt.Sprintf("%s/cosmos/tx/v1beta1/txs/%s", l.baseUrl, hash)
	response, err := l.client.Get(reqUrl)
	if err != nil {
		return nil, errors.Wrap(err, "lcdImpl.Tx")
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrap(err, "lcdImpl.Tx")
	}
	LcdTxRes := LcdTxRes{}

	if err := json.Unmarshal(data, &LcdTxRes); err != nil {
		return nil, errors.Wrap(err, "lcdImpl.Tx")
	}

	return &LcdTxRes, nil
}

func (l *lcdImpl) ContractState(address string, query string, height ...uint64) ([]byte, error) {
	reqUrl := fmt.Sprintf("%s/cosmwasm/wasm/v1/contract/%s/smart/%s", l.baseUrl, address, query)
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return nil, errors.Wrap(err, "lcdImpl.ContractState")
	}
	if len(height) > 0 {
		req.Header.Add(terra.LCD_BLOCK_HEIGHT_REQUEST_HEADER, strconv.FormatUint(height[0], 10))
	}

	response, err := l.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "lcdImpl.ContractState")
	}
	defer response.Body.Close()
	if len(height) > 0 {
		resHeight, err := strconv.ParseUint(response.Header.Get("Grpc-Metadata-X-Cosmos-Block-Height"), 10, 64)
		if err != nil {
			return nil, errors.Wrap(err, "lcdImpl.ContractState")
		}
		if resHeight != height[0] {
			return nil, errors.New(fmt.Sprintf("lcdImpl.ContractState: invalid height, expected %d, got %d", height[0], resHeight))
		}
	}

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrap(err, "lcdImpl.ContractState")
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
