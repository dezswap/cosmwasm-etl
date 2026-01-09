package terraswap

import (
	"errors"
	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/rpc"
	"github.com/stretchr/testify/assert"
)

var (
	mockRpcEventResWithSender = []rpc.RpcEventRes{
		{
			Type: string(eventlog.Message),
			Attributes: []rpc.RpcAttributeRes{
				{Key: "action", Value: "act"},
				{Key: "sender", Value: mockSender},
			},
		},
	}

	mockRpcEventRes = []rpc.RpcEventRes{
		{
			Type:       string(eventlog.Message),
			Attributes: []rpc.RpcAttributeRes{{Key: "action", Value: "act"}},
		},
	}
)

func TestPhoenixDataStore_convertLogToRawTx(t *testing.T) {
	blockTs := time.Now()
	r := &phoenixSourceDataStore{chainDataAdapter: &mockCda{sender: "shouldNotCall"}}
	tx, err := r.convertLogToRawTx("txhash", mockRpcEventResWithSender, blockTs)
	assert.NoError(t, err)
	assert.Equal(t, mockSender, tx.Sender)
	assert.Equal(t, "txhash", tx.Hash)
	assert.WithinDuration(t, blockTs, tx.Timestamp, time.Second)
	assert.NotEmpty(t, tx.LogResults)

}

func TestPhoenixDataStore_convertLogToRawTx_SenderFromCda(t *testing.T) {
	r := &phoenixSourceDataStore{chainDataAdapter: &mockCda{sender: "fromCDA"}}
	tx, err := r.convertLogToRawTx("txhash", mockRpcEventRes, time.Now())
	assert.NoError(t, err)
	assert.Equal(t, "fromCDA", tx.Sender)
}

func TestPhoenixDataStore_convertLogToRawTx_CdaReturnsErr(t *testing.T) {
	r := &phoenixSourceDataStore{chainDataAdapter: &mockCda{err: errors.New("fail")}}
	_, err := r.convertLogToRawTx("txhash", mockRpcEventRes, time.Now())
	assert.Error(t, err)
}

func TestPhoenixDataStore_groupLogAttrByType(t *testing.T) {
	events := []rpc.RpcEventRes{
		{
			Type:       "wasm",
			Attributes: []rpc.RpcAttributeRes{{Key: "k1", Value: "v1"}},
		},
		{
			Type:       "transfer",
			Attributes: []rpc.RpcAttributeRes{{Key: "k2", Value: "v2"}},
		},
		{
			Type:       "wasm",
			Attributes: []rpc.RpcAttributeRes{{Key: "k3", Value: "v3"}},
		},
	}
	r := &phoenixSourceDataStore{chainDataAdapter: &mockCda{sender: "fromCDA"}}
	result := r.groupLogAttrByType(events)
	assert.Len(t, result, 2)
	assert.Equal(t, eventlog.Attributes{{Key: "k2", Value: "v2"}}, result["transfer"])
	assert.ElementsMatch(t, []eventlog.Attribute{{Key: "k1", Value: "v1"}, {Key: "k3", Value: "v3"}}, result["wasm"])
}

func TestPhoenixDataStore_groupLogAttrByType_Empty(t *testing.T) {
	r := &phoenixSourceDataStore{chainDataAdapter: &mockCda{sender: "fromCDA"}}
	empty := r.groupLogAttrByType([]rpc.RpcEventRes{})
	assert.Empty(t, empty)
}

// https://finder.terra.money/mainnet/tx/D63A9704AA874A4DD89B642EEA7A49D903C248F01D4E40D9F0807E32DFE5D717
func TestPhoenixDataStore_groupLogAttrByType_D63A97(t *testing.T) {
	events := []rpc.RpcEventRes{
		{
			Type: "message",
			Attributes: []rpc.RpcAttributeRes{
				{Key: "action", Value: "/cosmwasm.wasm.v1.MsgExecuteContract"},
				{Key: "sender", Value: "terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3"},
				{Key: "module", Value: "wasm"},
			},
		},
		{
			Type:       "execute",
			Attributes: []rpc.RpcAttributeRes{{Key: "_contract_address", Value: "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"}},
		},
		{
			Type: "wasm",
			Attributes: []rpc.RpcAttributeRes{
				{Key: "_contract_address", Value: "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"},
				{Key: "action", Value: "provide_liquidity"},
				{Key: "sender", Value: "terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3"},
				{Key: "receiver", Value: "terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3"},
				{Key: "assets", Value: "1000000terra1ysd87nayjuelxj4wvp4wnp9as0mwszzkje6a9z6f3xx2903ghnsq4hm50y, 1000000terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"},
				{Key: "share", Value: "999000"},
				{Key: "refund_assets", Value: "0terra1ysd87nayjuelxj4wvp4wnp9as0mwszzkje6a9z6f3xx2903ghnsq4hm50y, 0terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"},
			},
		},
		{
			Type:       "execute",
			Attributes: []rpc.RpcAttributeRes{{Key: "_contract_address", Value: "terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut"}},
		},
		{
			Type: "wasm",
			Attributes: []rpc.RpcAttributeRes{
				{Key: "_contract_address", Value: "terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut"},
				{Key: "action", Value: "mint"},
				{Key: "amount", Value: "1000"},
				{Key: "to", Value: "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"},
			},
		},
		{
			Type:       "execute",
			Attributes: []rpc.RpcAttributeRes{{Key: "_contract_address", Value: "terra1ysd87nayjuelxj4wvp4wnp9as0mwszzkje6a9z6f3xx2903ghnsq4hm50y"}},
		},
		{
			Type: "wasm",
			Attributes: []rpc.RpcAttributeRes{
				{Key: "_contract_address", Value: "terra1ysd87nayjuelxj4wvp4wnp9as0mwszzkje6a9z6f3xx2903ghnsq4hm50y"},
				{Key: "action", Value: "transfer_from"},
				{Key: "amount", Value: "1000000"},
				{Key: "by", Value: "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"},
				{Key: "from", Value: "terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3"},
				{Key: "to", Value: "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"},
			},
		},
		{
			Type:       "execute",
			Attributes: []rpc.RpcAttributeRes{{Key: "_contract_address", Value: "terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"}},
		},
		{
			Type: "wasm",
			Attributes: []rpc.RpcAttributeRes{
				{Key: "_contract_address", Value: "terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"},
				{Key: "action", Value: "transfer_from"},
				{Key: "amount", Value: "1000000"},
				{Key: "by", Value: "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"},
				{Key: "from", Value: "terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3"},
				{Key: "to", Value: "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"},
			},
		},
		{
			Type:       "execute",
			Attributes: []rpc.RpcAttributeRes{{Key: "_contract_address", Value: "terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut"}},
		},
		{
			Type: "wasm",
			Attributes: []rpc.RpcAttributeRes{
				{Key: "_contract_address", Value: "terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut"},
				{Key: "action", Value: "mint"},
				{Key: "amount", Value: "999000"},
				{Key: "to", Value: "terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3"},
			},
		},
	}
	r := &phoenixSourceDataStore{chainDataAdapter: &mockCda{sender: "fromCDA"}}
	result := r.groupLogAttrByType(events)
	assert.Len(t, result, 3)
	assert.Len(t, result["message"], 3)
	assert.Len(t, result["execute"], 5)
	assert.Len(t, result["wasm"], 27)
}
