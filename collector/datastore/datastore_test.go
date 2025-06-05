package datastore

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	grpc1 "github.com/cosmos/gogoproto/grpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/types"

	wasm "github.com/CosmWasm/wasmd/x/wasm/types"
	tendermintType "github.com/cometbft/cometbft/proto/tendermint/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"

	"github.com/dezswap/cosmwasm-etl/configs"
	grpcConn "github.com/dezswap/cosmwasm-etl/pkg/grpc"
	"github.com/dezswap/cosmwasm-etl/pkg/s3client"
)

const startBlock int64 = 495466
const chainId string = "soju"

var testRawTxByte []byte
var testRawTxByte2 []byte
var testRawTxString string
var storeimpl *dataStoreImpl

func Test01GetChainId(t *testing.T) {
	receivedChainId := storeimpl.GetChainId()
	assert.Equal(t, chainId, receivedChainId)
}

func Test02GetLatestProcessedBlockNumber(t *testing.T) {
	mockS3Client := s3ClientMock{}
	mockS3Client.On("GetLatestProcessedBlockNumber", mock.Anything, mock.Anything).Return(startBlock, nil)
	mockS3ClientCreateFunc := func() (s3client.S3ClientInterface, error) {
		return &mockS3Client, nil
	}
	storeimpl.newS3ClientFunc = mockS3ClientCreateFunc

	blockNum, _ := storeimpl.GetLatestProcessedBlockNumber(chainId)
	assert.Equal(t, startBlock, blockNum)
}

func Test03GetBlockByHeight(t *testing.T) {
	m := serviceClientMock{}
	m.On("GetBlockWithTxs", mock.Anything, mock.Anything).Return(&txtypes.GetBlockWithTxsResponse{
		Block: &tendermintType.Block{},
	}, nil)
	mockFunc := func(cc grpc1.ClientConn) txtypes.ServiceClient {
		return &m
	}

	storeimpl.newServiceClientFunc = mockFunc
	resp, err := storeimpl.GetBlockByHeight(495465)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
}

func Test04extractTxMsgs(t *testing.T) {
	// test specific setup
	makeTxByte()

	m := serviceClientMock{}
	m.On("GetTx", mock.Anything, mock.Anything).Return(&txtypes.GetTxResponse{
		TxResponse: &types.TxResponse{},
	}, nil)

	mockFunc := func(cc grpc1.ClientConn) txtypes.ServiceClient {
		return &m
	}

	testBlock := &tendermintType.Block{
		Data: tendermintType.Data{
			Txs: [][]byte{testRawTxByte},
		},
	}

	storeimpl.newServiceClientFunc = mockFunc

	// test
	resp, err := storeimpl.extractTxMsgs(testBlock)

	assert.Nil(t, err)
	assert.NotNil(t, resp)
}

func Test05GetBlockTxsFromHeight(t *testing.T) {
	m := serviceClientMock{}
	m.On("GetBlockWithTxs", mock.Anything, mock.Anything).Return(&txtypes.GetBlockWithTxsResponse{
		Block: &tendermintType.Block{},
	}, nil)
	m.On("GetTx", mock.Anything, mock.Anything).Return(&txtypes.GetTxResponse{
		TxResponse: &types.TxResponse{},
	}, nil)
	mockFunc := func(cc grpc1.ClientConn) txtypes.ServiceClient {
		return &m
	}

	storeimpl.newServiceClientFunc = mockFunc

	resp, err := storeimpl.GetBlockTxsFromHeight(startBlock)

	assert.Nil(t, err)
	assert.NotNil(t, resp)
}

func Test06UnmarkLatestBlock(t *testing.T) {
	mockS3Client := s3ClientMock{}
	mockS3Client.On("UnmarkLatestBlock", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockS3ClientCreateFunc := func() (s3client.S3ClientInterface, error) {
		return &mockS3Client, nil
	}
	storeimpl.newS3ClientFunc = mockS3ClientCreateFunc

	err := storeimpl.ChangeLatestBlock(startBlock, chainId)
	assert.Nil(t, err)
}

func Test07UploadBlockBinaryAsLatest(t *testing.T) {
	mockS3Client := s3ClientMock{}
	mockS3Client.On("UploadBlockBinaryAsLatest", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockS3ClientCreateFunc := func() (s3client.S3ClientInterface, error) {
		return &mockS3Client, nil
	}
	storeimpl.newS3ClientFunc = mockS3ClientCreateFunc

	err := storeimpl.UploadBlockBinary(startBlock, []byte{}, chainId)
	assert.Nil(t, err)
}

func Test08GetCurrentPairsList(t *testing.T) {
	m := serviceClientMock{}

	m.On("SmartContractState",
		mock.Anything,
		&wasm.QuerySmartContractStateRequest{
			Address:   storeimpl.FactoryContractAddress,
			QueryData: wasm.RawContractMessage([]byte(`{"pairs":{"start_after":[{"token":{"contract_addr":"terra1spkm49wd9dqkranhrks4cupecl3rtgeqqljq3qrvrrts2ev2gw6sy5vz3k"}},{"native_token":{"denom":"uluna"}}]}}`)),
		}).Once().Return(&wasm.QuerySmartContractStateResponse{
		Data: wasm.RawContractMessage([]byte(`{"pairs":[]}`)),
	}, nil)

	m.On("SmartContractState", mock.Anything,
		&wasm.QuerySmartContractStateRequest{
			Address:   storeimpl.FactoryContractAddress,
			QueryData: wasm.RawContractMessage([]byte(`{"pairs":{}}`)),
		}).Once().Return(&wasm.QuerySmartContractStateResponse{
		Data: wasm.RawContractMessage([]byte(factoryResponse)),
	}, nil)

	mockFunc := func(cc grpc1.ClientConn) wasm.QueryClient {
		return &m
	}

	storeimpl.newQueryClientFunc = mockFunc

	ret, err := storeimpl.GetCurrentPairsList(startBlock)
	assert.NoError(t, err)

	expectedRet := &PairListDTO{}
	{
		rawExpectedRet := &PairListResponse{}
		err = json.Unmarshal([]byte(factoryResponse), rawExpectedRet)
		if err != nil {
			panic(err)
		}

		expectedRet.Pairs = make(map[string]UnitPairDTO)
		for _, unit := range rawExpectedRet.Pairs {
			expectedRet.Pairs[unit.ContractAddr] = *unit.Convert()
		}
	}

	assert.Equal(t, expectedRet, ret)
}

func Test09GetCurrentPoolStatusOfUnitPair(t *testing.T) {
	m := serviceClientMock{}

	m.On("SmartContractState", mock.Anything,
		&wasm.QuerySmartContractStateRequest{
			Address:   "",
			QueryData: wasm.RawContractMessage([]byte(`{"pairs": {}}`)),
		}).Return(&wasm.QuerySmartContractStateResponse{
		Data: wasm.RawContractMessage([]byte(factoryResponse)),
	}, nil)

	m.On("SmartContractState", mock.Anything,
		&wasm.QuerySmartContractStateRequest{
			Address:   "terra1xjv2pmf26yaz3wqft7caafgckdg4eflzsw56aqhdcjw58qx0v2mqux87t8",
			QueryData: wasm.RawContractMessage([]byte(`{"pool": {}}`)),
		}).Return(&wasm.QuerySmartContractStateResponse{
		Data: wasm.RawContractMessage([]byte(poolResponse)),
	}, nil)

	mockFunc := func(cc grpc1.ClientConn) wasm.QueryClient {
		return &m
	}

	storeimpl.newQueryClientFunc = mockFunc

	ret, err := storeimpl.GetCurrentPoolStatusOfUnitPair(startBlock, "terra1xjv2pmf26yaz3wqft7caafgckdg4eflzsw56aqhdcjw58qx0v2mqux87t8")
	assert.NoError(t, err)

	expectedRet := &PoolInfo{}
	err = json.Unmarshal([]byte(poolResponse), expectedRet)
	if err != nil {
		panic(err)
	}

	assert.Equal(t, expectedRet.Convert(), ret)
}

func Test10GetCurrentPoolStatusOfAllPairs(t *testing.T) {
	m := serviceClientMock{}

	m.On("SmartContractState",
		mock.Anything,
		&wasm.QuerySmartContractStateRequest{
			Address:   "",
			QueryData: []byte(`{"pairs":{"start_after":[{"token":{"contract_addr":"terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"}},{"native_token":{"denom":"uluna"}}]}}`),
		}).Once().Return(&wasm.QuerySmartContractStateResponse{
		Data: []byte(`{"pairs":[]}`),
	}, nil)
	m.On("SmartContractState", mock.Anything,
		&wasm.QuerySmartContractStateRequest{
			Address:   "",
			QueryData: []byte(`{"pairs":{}}`),
		}).Once().Return(&wasm.QuerySmartContractStateResponse{
		Data: []byte(lightFactoryResp),
	}, nil)

	m.On("SmartContractState", mock.Anything,
		&wasm.QuerySmartContractStateRequest{
			Address:   "terra1xjv2pmf26yaz3wqft7caafgckdg4eflzsw56aqhdcjw58qx0v2mqux87t8",
			QueryData: []byte(`{"pool": {}}`),
		}).Return(&wasm.QuerySmartContractStateResponse{
		Data: []byte(poolResponse),
	}, nil)

	mockFunc := func(cc grpc1.ClientConn) wasm.QueryClient {
		return &m
	}

	storeimpl.newQueryClientFunc = mockFunc

	ret, err := storeimpl.GetCurrentPoolStatusOfAllPairs(startBlock)
	assert.NoError(t, err)

	expectedRet := &PoolInfoList{}
	err = json.Unmarshal([]byte(lightWholePoolResp), expectedRet)
	if err != nil {
		panic(err)
	}

	assert.Equal(t, expectedRet, ret)
}

func Test11GetPoolStatusOfUnitPairByHeight(t *testing.T) {
	mockS3Client := s3ClientMock{}
	mockS3Client.On("GetFileFromS3", mock.Anything).Return([]byte(lightWholePoolResp), nil)
	mockS3ClientCreateFunc := func() (s3client.S3ClientInterface, error) {
		return &mockS3Client, nil
	}

	storeimpl.newS3ClientFunc = mockS3ClientCreateFunc

	path := []string{"collector", "pairs"}
	ret, err := storeimpl.GetPoolStatusOfUnitPairByHeight(111, "terra1xjv2pmf26yaz3wqft7caafgckdg4eflzsw56aqhdcjw58qx0v2mqux87t8", path...)
	assert.NoError(t, err)

	expectedRet := &PoolInfoWithLpAddr{}
	err = json.Unmarshal([]byte(unitPoolResp), expectedRet)
	if err != nil {
		panic(err)
	}

	assert.Equal(t, expectedRet, ret)
}

func Test12GetPoolStatusOfAllPairsByHeight(t *testing.T) {
	mockS3Client := s3ClientMock{}
	mockS3Client.On("GetFileFromS3", mock.Anything).Return([]byte(poolStatusFromS3), nil)
	mockS3ClientCreateFunc := func() (s3client.S3ClientInterface, error) {
		return &mockS3Client, nil
	}

	storeimpl.newS3ClientFunc = mockS3ClientCreateFunc

	path := []string{"collector", "pairs"}
	ret, err := storeimpl.GetPoolStatusOfAllPairsByHeight(111, path...)
	assert.NoError(t, err)

	expectedRet := &PoolInfoList{}
	err = json.Unmarshal([]byte(poolStatusFromS3), expectedRet)
	if err != nil {
		panic(err)
	}

	assert.Equal(t, expectedRet, ret)
}

func Test13UploadPoolInfoBinary(t *testing.T) {
	mockS3Client := s3ClientMock{}
	mockS3Client.On("UploadFileToS3", mock.Anything, mock.Anything).Return(nil)
	mockS3ClientCreateFunc := func() (s3client.S3ClientInterface, error) {
		return &mockS3Client, nil
	}

	storeimpl.newS3ClientFunc = mockS3ClientCreateFunc

	path := []string{"pairs"}
	err := storeimpl.UploadPoolInfoBinary(111, []byte(lightWholePoolResp), path...)

	assert.NoError(t, err)
}

func TestMain(m *testing.M) {
	setUp()
	code := m.Run()
	tearDown()
	os.Exit(code)
}

func setUp() {
	testconf := configs.New()
	serviceDesc := grpcConn.ServiceDescMock{}
	serviceDesc.On("GetConnection", mock.Anything).Return(&grpc.ClientConn{})
	lcdMock := lcdClientMock{}

	storeimplTemp, _ := New(testconf, &serviceDesc, &lcdMock)
	storeimpl = storeimplTemp.(*dataStoreImpl)

	time.Sleep(time.Second * 1)

	storeimpl.initInterfaceRegistry()
	storeimpl.chainId = chainId
	storeimpl.FactoryContractAddress = testconf.Collector.PairFactoryContractAddress

	// dummy add for testing AddCustomInterfaceRegistry
	storeimpl.AddCustomInterfaceRegistry(cryptocodec.RegisterInterfaces)
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
	args := s.MethodCalled("GetLatestProcessedBlockNumber", folderPath)
	return args.Get(0).(int64), args.Error(1)
}

func (s *s3ClientMock) ChangeLatestBlock(blockNum int64, folderPath ...string) error {
	args := s.MethodCalled("UnmarkLatestBlock", blockNum, folderPath)
	return args.Error(0)
}

func (s *s3ClientMock) UploadBlockBinary(blockNum int64, data []byte, folderPath ...string) error {
	args := s.MethodCalled("UploadBlockBinaryAsLatest", blockNum, data, folderPath)
	return args.Error(0)
}

func (s *s3ClientMock) GetFileFromS3(in ...string) ([]byte, error) {
	args := s.MethodCalled("GetFileFromS3", in)
	return args.Get(0).([]byte), args.Error(1)
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
