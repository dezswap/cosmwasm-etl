package cosmos45

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/dezswap/cosmwasm-etl/pkg/httpclient"
	"github.com/dezswap/cosmwasm-etl/pkg/nodeerr"
	"github.com/dezswap/cosmwasm-etl/pkg/terra"
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

	reqUrl := fmt.Sprintf("%s/cosmos/tx/v1beta1/txs/%s", l.baseUrl, hash)
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

	reqUrl := fmt.Sprintf("%s/cosmwasm/wasm/v1/contract/%s/smart/%s", l.baseUrl, address, query)
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return nil, errors.Wrap(err, op)
	}
	if len(height) > 0 {
		req.Header.Add(terra.LCD_BLOCK_HEIGHT_REQUEST_HEADER, strconv.FormatUint(height[0], 10))
	}

	response, err := l.client.Do(req)
	if err != nil {
		return nil, nodeerr.Transient(op, nodeerr.TransportLCD, l.host, err)
	}
	defer response.Body.Close()

	// Status first: a non-200 carries no height header, so checking the header before
	// the status turns every gateway failure into a strconv error that names neither
	// the status nor the node, and that no longer classifies as retryable.
	data, err := httpclient.ReadResponse(op, nodeerr.TransportLCD, l.host, response)
	if err != nil {
		return nil, err
	}

	if len(height) > 0 {
		resHeight, err := strconv.ParseUint(response.Header.Get("Grpc-Metadata-X-Cosmos-Block-Height"), 10, 64)
		if err != nil {
			return nil, errors.Wrap(err, op)
		}
		if resHeight != height[0] {
			return nil, errors.Errorf("%s: invalid height, expected %d, got %d", op, height[0], resHeight)
		}
	}

	return data, nil
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
