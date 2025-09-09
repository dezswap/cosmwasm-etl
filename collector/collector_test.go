package collector

import (
	context "context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"
	"time"

	grpc1 "github.com/cosmos/gogoproto/grpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	wasm "github.com/CosmWasm/wasmd/x/wasm/types"
	tendermintType "github.com/cometbft/cometbft/proto/tendermint/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"

	"github.com/dezswap/cosmwasm-etl/collector/datastore"
	"github.com/dezswap/cosmwasm-etl/configs"
	grpcConn "github.com/dezswap/cosmwasm-etl/pkg/grpc"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/dezswap/cosmwasm-etl/pkg/s3client"
)

const startBlock int64 = 495466

var (
	testRawTxByte   []byte
	testRawTxByte2  []byte
	testRawTxString string
	store           datastore.DataStore
	testFactoryAddr string
)

func Test01DoCollect(t *testing.T) {
	// test specific setup
	makeTxByte()
	logger := logging.New("test", configs.LogConfig{})

	m := serviceClientMock{}
	lcdMock := lcdClientMock{}

	// collect block
	// input 2 valid block & 1 pending response
	m.On("GetBlockWithTxs", mock.Anything, &txtypes.GetBlockWithTxsRequest{Height: startBlock}).
		Return(&txtypes.GetBlockWithTxsResponse{
			Block: &tendermintType.Block{
				Header: tendermintType.Header{
					Height: startBlock,
				},
				Data: tendermintType.Data{
					Txs: [][]byte{testRawTxByte},
				},
			},
		}, nil)
	m.On("GetBlockWithTxs", mock.Anything, &txtypes.GetBlockWithTxsRequest{Height: startBlock + 1}).
		Return(&txtypes.GetBlockWithTxsResponse{
			Block: &tendermintType.Block{
				Header: tendermintType.Header{
					Height: startBlock + 1,
				},
				Data: tendermintType.Data{
					Txs: [][]byte{testRawTxByte2},
				},
			},
		}, nil)
	m.On("GetBlockWithTxs", mock.Anything, &txtypes.GetBlockWithTxsRequest{Height: startBlock + 2}).
		Return((*txtypes.GetBlockWithTxsResponse)(nil), fmt.Errorf("invalid height: %d", startBlock+2))

	// process block
	m.On("GetTx", mock.Anything, &txtypes.GetTxRequest{Hash: "1608CE8E8C3AC55FF7F04A3C5771B6732B84AF9205F4788A607B1CB05125AC3B"}).
		Return(&txtypes.GetTxResponse{
			Tx: &txtypes.Tx{},
			TxResponse: &types.TxResponse{
				Height: startBlock,
				TxHash: "1608CE8E8C3AC55FF7F04A3C5771B6732B84AF9205F4788A607B1CB05125AC3B",
				Code:   0,
				RawLog: testRawTxString,
			},
		}, nil)

	m.On("GetTx", mock.Anything, &txtypes.GetTxRequest{Hash: "5EC8E130FBD2FFAAC87252C2E10AF4AF67900D9866F9ED053FC6A2137E559CD2"}).
		Return(&txtypes.GetTxResponse{
			Tx: &txtypes.Tx{},
			TxResponse: &types.TxResponse{
				Height: startBlock + 1,
				TxHash: "5EC8E130FBD2FFAAC87252C2E10AF4AF67900D9866F9ED053FC6A2137E559CD2",
				Code:   0,
				RawLog: testRawTxString,
			},
		}, nil)

	// querying pairs
	m.On("SmartContractState",
		mock.Anything,
		&wasm.QuerySmartContractStateRequest{
			Address:   testFactoryAddr,
			QueryData: wasm.RawContractMessage([]byte(`{"pairs":{"start_after":[{"token":{"contract_addr":"terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"}},{"native_token":{"denom":"uluna"}}]}}`)),
		}).Once().Return(&wasm.QuerySmartContractStateResponse{
		Data: wasm.RawContractMessage([]byte(emptyFactoryPairsRes)),
	}, nil)

	m.On("SmartContractState", mock.Anything,
		&wasm.QuerySmartContractStateRequest{
			Address:   testFactoryAddr,
			QueryData: wasm.RawContractMessage([]byte(`{"pairs":{}}`)),
		}).Once().Return(&wasm.QuerySmartContractStateResponse{
		Data: wasm.RawContractMessage([]byte(lightFactoryResp)),
	}, nil)

	m.On("SmartContractState", mock.Anything,
		&wasm.QuerySmartContractStateRequest{
			Address:   "terra1xjv2pmf26yaz3wqft7caafgckdg4eflzsw56aqhdcjw58qx0v2mqux87t8",
			QueryData: wasm.RawContractMessage([]byte(`{"pool": {}}`)),
		}).Return(&wasm.QuerySmartContractStateResponse{
		Data: wasm.RawContractMessage([]byte(poolResponse)),
	}, nil)

	mockFunc := func(cc grpc1.ClientConn) txtypes.ServiceClient {
		return &m
	}

	mockQueryFunc := func(cc grpc1.ClientConn) wasm.QueryClient {
		return &m
	}

	store.SetNewServiceClientFunc(mockFunc)
	store.SetNewQueryClientFunc(mockQueryFunc)

	mockS3Client := s3ClientMock{}

	// collect block
	// the latest old block
	mockS3Client.On("GetLatestProcessedBlockNumber", mock.Anything).Return(startBlock-1, nil)

	// process block
	mockS3Client.On("ChangeLatestBlock", int64(startBlock-1), mock.Anything).Return(nil)
	mockS3Client.On("ChangeLatestBlock", int64(startBlock), mock.Anything).Return(fmt.Errorf("artificial error")) // <- for finishing test
	mockS3Client.On("UploadBlockBinary", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockS3Client.On("UploadFileToS3", mock.Anything, mock.Anything).Return(nil)

	lcdMock.On("GetTx", mock.Anything).Return(nil, nil)

	mockS3ClientCreateFunc := func() (s3client.S3ClientInterface, error) {
		return &mockS3Client, nil
	}

	store.SetNewS3ClientFunc(mockS3ClientCreateFunc)

	// run test method
	err := DoCollect(store, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "artificial error")
}

func TestMain(m *testing.M) {
	setUp()
	code := m.Run()
	tearDown()
	os.Exit(code)
}

func setUp() {
	var err error
	testconf := configs.New()
	testFactoryAddr = testconf.Collector.PairFactoryContractAddress
	testServiceDesc := grpcConn.ServiceDescMock{}
	testServiceDesc.On("GetConnection", mock.Anything).Return(&grpc.ClientConn{})
	lcdMock := lcdClientMock{}
	store, err = datastore.New(testconf, &testServiceDesc, &lcdMock)
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Second * 1)

	// dummy add for testing AddCustomInterfaceRegistry
	store.AddCustomInterfaceRegistry(cryptocodec.RegisterInterfaces)
}

func tearDown() {}

type serviceClientMock struct {
	mock.Mock
}

var _ txtypes.ServiceClient = &serviceClientMock{}
var _ wasm.QueryClient = &serviceClientMock{}

// BroadcastTx implements tx.ServiceClient
func (s *serviceClientMock) BroadcastTx(ctx context.Context, in *txtypes.BroadcastTxRequest, opts ...grpc.CallOption) (*txtypes.BroadcastTxResponse, error) {
	args := s.MethodCalled("BroadcastTx", ctx, in)
	return args.Get(0).(*txtypes.BroadcastTxResponse), args.Error(1)
}

// GetBlockWithTxs implements tx.ServiceClient
func (s *serviceClientMock) GetBlockWithTxs(ctx context.Context, in *txtypes.GetBlockWithTxsRequest, opts ...grpc.CallOption) (*txtypes.GetBlockWithTxsResponse, error) {
	args := s.MethodCalled("GetBlockWithTxs", ctx, in)
	return args.Get(0).(*txtypes.GetBlockWithTxsResponse), args.Error(1)
}

// GetTx implements tx.ServiceClient
func (s *serviceClientMock) GetTx(ctx context.Context, in *txtypes.GetTxRequest, opts ...grpc.CallOption) (*txtypes.GetTxResponse, error) {
	args := s.MethodCalled("GetTx", ctx, in)
	return args.Get(0).(*txtypes.GetTxResponse), args.Error(1)
}

// GetTxsEvent implements tx.ServiceClient
func (s *serviceClientMock) GetTxsEvent(ctx context.Context, in *txtypes.GetTxsEventRequest, opts ...grpc.CallOption) (*txtypes.GetTxsEventResponse, error) {
	args := s.MethodCalled("GetTxsEvent", ctx, in)
	return args.Get(0).(*txtypes.GetTxsEventResponse), args.Error(1)
}

// Simulate implements tx.ServiceClient
func (s *serviceClientMock) Simulate(ctx context.Context, in *txtypes.SimulateRequest, opts ...grpc.CallOption) (*txtypes.SimulateResponse, error) {
	args := s.MethodCalled("Simulate", ctx, in)
	return args.Get(0).(*txtypes.SimulateResponse), args.Error(1)
}

func (s *serviceClientMock) ContractInfo(ctx context.Context, in *wasm.QueryContractInfoRequest, opts ...grpc.CallOption) (*wasm.QueryContractInfoResponse, error) {
	panic("unimplemented")
}

func (s *serviceClientMock) ContractHistory(ctx context.Context, in *wasm.QueryContractHistoryRequest, opts ...grpc.CallOption) (*wasm.QueryContractHistoryResponse, error) {
	panic("unimplemented")
}

func (s *serviceClientMock) ContractsByCode(ctx context.Context, in *wasm.QueryContractsByCodeRequest, opts ...grpc.CallOption) (*wasm.QueryContractsByCodeResponse, error) {
	panic("unimplemented")
}

func (s *serviceClientMock) AllContractState(ctx context.Context, in *wasm.QueryAllContractStateRequest, opts ...grpc.CallOption) (*wasm.QueryAllContractStateResponse, error) {
	panic("unimplemented")
}

func (s *serviceClientMock) RawContractState(ctx context.Context, in *wasm.QueryRawContractStateRequest, opts ...grpc.CallOption) (*wasm.QueryRawContractStateResponse, error) {
	panic("unimplemented")
}

// Simulate implements wasm.QueryClient
func (s *serviceClientMock) SmartContractState(ctx context.Context, in *wasm.QuerySmartContractStateRequest, opts ...grpc.CallOption) (*wasm.QuerySmartContractStateResponse, error) {
	args := s.MethodCalled("SmartContractState", ctx, in)
	return args.Get(0).(*wasm.QuerySmartContractStateResponse), args.Error(1)
}

func (s *serviceClientMock) Code(ctx context.Context, in *wasm.QueryCodeRequest, opts ...grpc.CallOption) (*wasm.QueryCodeResponse, error) {
	panic("unimplemented")
}

func (s *serviceClientMock) Codes(ctx context.Context, in *wasm.QueryCodesRequest, opts ...grpc.CallOption) (*wasm.QueryCodesResponse, error) {
	panic("unimplemented")
}

func (s *serviceClientMock) PinnedCodes(ctx context.Context, in *wasm.QueryPinnedCodesRequest, opts ...grpc.CallOption) (*wasm.QueryPinnedCodesResponse, error) {
	panic("unimplemented")
}

func (s *serviceClientMock) Params(ctx context.Context, in *wasm.QueryParamsRequest, opts ...grpc.CallOption) (*wasm.QueryParamsResponse, error) {
	panic("implement me")
}

func (s *serviceClientMock) ContractsByCreator(ctx context.Context, in *wasm.QueryContractsByCreatorRequest, opts ...grpc.CallOption) (*wasm.QueryContractsByCreatorResponse, error) {
	panic("implement me")
}

func (s *serviceClientMock) BuildAddress(ctx context.Context, in *wasm.QueryBuildAddressRequest, opts ...grpc.CallOption) (*wasm.QueryBuildAddressResponse, error) {
	panic("implement me")
}

func (s *serviceClientMock) TxDecode(ctx context.Context, in *txtypes.TxDecodeRequest, opts ...grpc.CallOption) (*txtypes.TxDecodeResponse, error) {
	panic("implement me")
}

func (s *serviceClientMock) TxEncode(ctx context.Context, in *txtypes.TxEncodeRequest, opts ...grpc.CallOption) (*txtypes.TxEncodeResponse, error) {
	panic("implement me")
}

func (s *serviceClientMock) TxEncodeAmino(ctx context.Context, in *txtypes.TxEncodeAminoRequest, opts ...grpc.CallOption) (*txtypes.TxEncodeAminoResponse, error) {
	panic("implement me")
}

func (s *serviceClientMock) TxDecodeAmino(ctx context.Context, in *txtypes.TxDecodeAminoRequest, opts ...grpc.CallOption) (*txtypes.TxDecodeAminoResponse, error) {
	panic("implement me")
}

type s3ClientMock struct {
	mock.Mock
}

var _ s3client.S3ClientInterface = &s3ClientMock{}

func (s *s3ClientMock) GetLatestProcessedBlockNumber(folderPath ...string) (int64, error) {
	args := s.MethodCalled("GetLatestProcessedBlockNumber", mock.Anything)
	return args.Get(0).(int64), args.Error(1)
}

func (s *s3ClientMock) ChangeLatestBlock(blockNum int64, folderPath ...string) error {
	args := s.MethodCalled("ChangeLatestBlock", blockNum, folderPath)
	return args.Error(0)
}

func (s *s3ClientMock) UploadBlockBinary(blockNum int64, data []byte, folderPath ...string) error {
	args := s.MethodCalled("UploadBlockBinary", blockNum, data, folderPath)
	return args.Error(0)
}

func (s *s3ClientMock) GetFileFromS3(in ...string) ([]byte, error) {
	args := s.MethodCalled("GetFileFromS3", in)
	return args.Get(0).([]byte), args.Error(0)
}

func (s *s3ClientMock) UploadFileToS3(in1 []byte, in2 ...string) error {
	args := s.MethodCalled("UploadFileToS3", in1, in2)
	return args.Error(0)
}

func (s *s3ClientMock) GetBlockFilePath(blockNum int64, folderPath ...string) []string {
	return append(folderPath, fmt.Sprintf("%s_%d.json", "block", blockNum))
}

func makeTxByte() {
	var err error
	testRawTxByte, err = base64.StdEncoding.DecodeString("CqgCCqUCCiQvY29zbXdhc20ud2FzbS52MS5Nc2dFeGVjdXRlQ29udHJhY3QS/AEKLHRlcnJhMTR2ajZlZDRoZ203ZHY5NGR6NzY5NjRnM2x4bDV3ajk1amFmcGw4EkB0ZXJyYTF6NzcwNXQycDVwNnJlbDkzZmQ3enJzaDhhNGx1eHl4ejg4YTR6a21sY3R3ZjM4eWg1MjBxdDk0OW44GnZ7InN3YXAiOnsibWluaW11bV9yZWNlaXZlIjoiOTg1MTQyMTkiLCJvZmZlcl9hc3NldCI6eyJhbW91bnQiOiIxMDAwMDAwMDAiLCJpbmZvIjp7Im5hdGl2ZV90b2tlbiI6eyJkZW5vbSI6InVsdW5hIn19fX19KhIKBXVsdW5hEgkxMDAwMDAwMDASaApQCkYKHy9jb3Ntb3MuY3J5cHRvLnNlY3AyNTZrMS5QdWJLZXkSIwohA4RvoQ6AJPcezRNaBc8IK6qS0iTJUsikM6AuVJfGLCfdEgQKAggBGBcSFAoOCgV1bHVuYRIFOTAxNDUQg9ckGkD7+6yMOW70wuuXu1tIcMBBOGHYnY1mBbeBI5AJcK3k6CfobItfYil9pJcxpkvZ33Jxlk3r6xISEJDpfGTi0Evc")
	if err != nil {
		panic(err)
	}

	testRawTxByte2, err = base64.StdEncoding.DecodeString("CqMBCqABCiUvY29zbW9zLnN0YWtpbmcudjFiZXRhMS5Nc2dVbmRlbGVnYXRlEncKLHRlcnJhMWhkbXg3ODBkc2ttZDQ1dHU3NmZoa2tnc3d5NHZlcjN5OTh3d2d4EjN0ZXJyYXZhbG9wZXIxa2duNnp6bmZxMnh3cnczaHJoN3F0cWp3dTk4ZXM3dnQ4YXpqbDMaEgoFdWx1bmESCTEwOTMyMDUyMBJmCk4KRgofL2Nvc21vcy5jcnlwdG8uc2VjcDI1NmsxLlB1YktleRIjCiEDdHnIdCP9Dyy3zyp/VVPZ+1Qw1jVCROJ2IBGsbMQpp/8SBAoCCAESFAoOCgV1bHVuYRIFNTc4NjgQ+sUXGkCc2s5SVR/yQeT3JpTCEzERCFYeLRxmSFvgeLXm8hWX1z7+zzrrzJxhoKWjgyPhsgZ6yCG9qQ6B0pp3niiM0Qbb")
	if err != nil {
		panic(err)
	}

	testRawTxString = `"[{\"events\":[{\"type\":\"coin_received\",\"attributes\":[{\"key\":\"receiver\",\"value\":\"terra1z7705t2p5p6rel93fd7zrsh8a4luxyxz88a4zkmlctwf38yh520qt949n8\"},{\"key\":\"amount\",\"value\":\"100000000uluna\"}]},{\"type\":\"coin_spent\",\"attributes\":[{\"key\":\"spender\",\"value\":\"terra14vj6ed4hgm7dv94dz76964g3lxl5wj95jafpl8\"},{\"key\":\"amount\",\"value\":\"100000000uluna\"}]},{\"type\":\"execute\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra1z7705t2p5p6rel93fd7zrsh8a4luxyxz88a4zkmlctwf38yh520qt949n8\"},{\"key\":\"_contract_address\",\"value\":\"terra14xsm2wzvu7xaf567r693vgfkhmvfs08l68h4tjj5wjgyn5ky8e2qvzyanh\"}]},{\"type\":\"message\",\"attributes\":[{\"key\":\"action\",\"value\":\"/cosmwasm.wasm.v1.MsgExecuteContract\"},{\"key\":\"module\",\"value\":\"wasm\"},{\"key\":\"sender\",\"value\":\"terra14vj6ed4hgm7dv94dz76964g3lxl5wj95jafpl8\"}]},{\"type\":\"transfer\",\"attributes\":[{\"key\":\"recipient\",\"value\":\"terra1z7705t2p5p6rel93fd7zrsh8a4luxyxz88a4zkmlctwf38yh520qt949n8\"},{\"key\":\"sender\",\"value\":\"terra14vj6ed4hgm7dv94dz76964g3lxl5wj95jafpl8\"},{\"key\":\"amount\",\"value\":\"100000000uluna\"}]},{\"type\":\"wasm\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra1z7705t2p5p6rel93fd7zrsh8a4luxyxz88a4zkmlctwf38yh520qt949n8\"},{\"key\":\"action\",\"value\":\"swap\"},{\"key\":\"sender\",\"value\":\"terra14vj6ed4hgm7dv94dz76964g3lxl5wj95jafpl8\"},{\"key\":\"receiver\",\"value\":\"terra14vj6ed4hgm7dv94dz76964g3lxl5wj95jafpl8\"},{\"key\":\"offer_asset\",\"value\":\"uluna\"},{\"key\":\"ask_asset\",\"value\":\"terra14xsm2wzvu7xaf567r693vgfkhmvfs08l68h4tjj5wjgyn5ky8e2qvzyanh\"},{\"key\":\"offer_amount\",\"value\":\"100000000\"},{\"key\":\"return_amount\",\"value\":\"99009265\"},{\"key\":\"spread_amount\",\"value\":\"951116\"},{\"key\":\"commission_amount\",\"value\":\"39619\"},{\"key\":\"maker_fee_amount\",\"value\":\"0\"},{\"key\":\"_contract_address\",\"value\":\"terra14xsm2wzvu7xaf567r693vgfkhmvfs08l68h4tjj5wjgyn5ky8e2qvzyanh\"},{\"key\":\"action\",\"value\":\"transfer\"},{\"key\":\"from\",\"value\":\"terra1z7705t2p5p6rel93fd7zrsh8a4luxyxz88a4zkmlctwf38yh520qt949n8\"},{\"key\":\"to\",\"value\":\"terra14vj6ed4hgm7dv94dz76964g3lxl5wj95jafpl8\"},{\"key\":\"amount\",\"value\":\"99009265\"}]}]}]",`
}

const (
	poolResponse = `
		{
			"assets": [
				{
					"info": {
						"token": {
							"contract_addr": "terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"
						}
					},
					"amount": "50000000000000"
				},
				{
					"info": {
						"native_token": {
							"denom": "uluna"
						}
					},
					"amount": "5000000"
				}
			],
			"total_share": "15811388300"
		}
	`

	lightFactoryResp = `
		{
			"pairs": [
				{
					"asset_infos": [
						{
							"token": {
								"contract_addr": "terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"
							}
						},
						{
							"native_token": {
								"denom": "uluna"
							}
						}
					],
					"contract_addr": "terra1xjv2pmf26yaz3wqft7caafgckdg4eflzsw56aqhdcjw58qx0v2mqux87t8",
					"liquidity_token": "terra1ttaekgrc60xc3xcflq069m49lwu79m5t552rjcws48rzhxcr4g6shmdw2v",
					"asset_decimals": [
						6,
						6
					]
				}
			]
		}
	`
	emptyFactoryPairsRes = `{"pairs": []}`
)

type lcdClientMock struct {
	mock.Mock
}

// GetTx implements lcdClient
func (c *lcdClientMock) GetTx(txHash string) (*txtypes.GetTxResponse, error) {
	args := c.Mock.MethodCalled("GetTx", txHash)
	return args.Get(0).(*txtypes.GetTxResponse), args.Error(1)
}

// GetBlockWithTxs implements lcdClient.
func (c *lcdClientMock) GetBlockWithTxs(height int64) (*txtypes.GetBlockWithTxsResponse, error) {
	args := c.Mock.MethodCalled("GetBlockWithTxs", height)
	return args.Get(0).(*txtypes.GetBlockWithTxsResponse), args.Error(1)
}
