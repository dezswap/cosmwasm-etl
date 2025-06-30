package terraswap

import (
	"errors"
	"testing"

	"github.com/dezswap/cosmwasm-etl/pkg/terra/cosmos45"
	"github.com/stretchr/testify/assert"
)

type mockCol5Lcd struct {
	TxFn func(hash string) (*cosmos45.LcdTxRes, error)
}

func (m *mockCol5Lcd) ContractState(address string, query string, height ...uint64) ([]byte, error) {
	panic("not implemented")
}
func (m *mockCol5Lcd) Tx(hash string) (*cosmos45.LcdTxRes, error) {
	return m.TxFn(hash)
}

func TestCol5ChainDataAdapter_TxSenderOf(t *testing.T) {
	adapter := &col5ChainDataAdapter{
		lcd: &mockCol5Lcd{
			TxFn: func(hash string) (*cosmos45.LcdTxRes, error) {
				if hash == "err" {
					return nil, errors.New("some error")
				}
				if hash == "nosender" {
					return &cosmos45.LcdTxRes{
						Tx: cosmos45.LcdTx{
							Body: cosmos45.LcdTxBody{
								Messages: []cosmos45.LcdTxMessage{{
									Type:   "/cosmwasm.wasm.v1.MsgExecuteContract",
									Sender: "",
								}},
							},
						},
					}, nil
				}
				if hash == "othermsg" {
					return &cosmos45.LcdTxRes{
						Tx: cosmos45.LcdTx{
							Body: cosmos45.LcdTxBody{
								Messages: []cosmos45.LcdTxMessage{{
									Type:   "/cosmos.bank.v1beta1.MsgSend",
									Sender: "shouldnotreturn",
								}},
							},
						},
					}, nil
				}
				return &cosmos45.LcdTxRes{
					Tx: cosmos45.LcdTx{
						Body: cosmos45.LcdTxBody{
							Messages: []cosmos45.LcdTxMessage{{
								Type:   "/cosmwasm.wasm.v1.MsgExecuteContract",
								Sender: "cosmos1sender",
							}},
						},
					},
				}, nil
			},
		},
	}

	t.Run("success", func(t *testing.T) {
		sender, err := adapter.TxSenderOf("txhash")
		assert.NoError(t, err)
		assert.Equal(t, "cosmos1sender", sender)
	})

	t.Run("error from lcd", func(t *testing.T) {
		sender, err := adapter.TxSenderOf("err")
		assert.Error(t, err)
		assert.Empty(t, sender)
	})

	t.Run("no wasm msg", func(t *testing.T) {
		sender, err := adapter.TxSenderOf("othermsg")
		assert.NoError(t, err)
		assert.Empty(t, sender)
	})

	t.Run("wasm msg but no sender", func(t *testing.T) {
		sender, err := adapter.TxSenderOf("nosender")
		assert.NoError(t, err)
		assert.Empty(t, sender)
	})
}
