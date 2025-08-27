package terraswap

import (
	"errors"
	"testing"

	"github.com/dezswap/cosmwasm-etl/pkg/terra/col4"
	"github.com/stretchr/testify/assert"
)

type mockCol4Lcd struct {
	TxFn func(hash string) (*col4.LcdTxRes, error)
}

func (m *mockCol4Lcd) ContractState(address string, query string, height ...uint64) ([]byte, error) {
	panic("not implemented")
}
func (m *mockCol4Lcd) Tx(hash string) (*col4.LcdTxRes, error) {
	return m.TxFn(hash)
}

func TestCol4ChainDataAdapter_TxSenderOf(t *testing.T) {
	adapter := &col4ChainDataAdapter{
		lcd: &mockCol4Lcd{
			TxFn: func(hash string) (*col4.LcdTxRes, error) {
				if hash == "err" {
					return nil, errors.New("some error")
				}
				if hash == "nosender" {
					return &col4.LcdTxRes{
						Tx: col4.TxRes{
							Value: col4.TxValueRes{
								Msg: []col4.MsgRes{{
									Type:  "wasm/MsgExecuteContract",
									Value: col4.WasmMsgValueRes{Sender: ""},
								}},
							},
						},
					}, nil
				}
				if hash == "othermsg" {
					return &col4.LcdTxRes{
						Tx: col4.TxRes{
							Value: col4.TxValueRes{
								Msg: []col4.MsgRes{{
									Type:  "bank/MsgSend",
									Value: col4.WasmMsgValueRes{Sender: "shouldnotreturn"},
								}},
							},
						},
					}, nil
				}
				return &col4.LcdTxRes{
					Tx: col4.TxRes{
						Value: col4.TxValueRes{
							Msg: []col4.MsgRes{{
								Type:  col4.LCD_TERRA_TX_MSG_WASM_TYPE,
								Value: col4.WasmMsgValueRes{Sender: "terra1sender"},
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
		assert.Equal(t, "terra1sender", sender)
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
