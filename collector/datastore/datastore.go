package datastore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	grpc1 "github.com/gogo/protobuf/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	"github.com/pkg/errors"
	tendermintType "github.com/tendermint/tendermint/proto/tendermint/types"

	// cosmos.base.tendermint.v1beta1
	wasm "github.com/CosmWasm/wasmd/x/wasm/types"
	tmservice "github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	cosmossdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	authz "github.com/cosmos/cosmos-sdk/x/authz" // will be replaced into x/authz/types
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"

	// v1beta1 "github.com/cosmos/cosmos-sdk/x/base/"
	crisis "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distribution "github.com/cosmos/cosmos-sdk/x/distribution/types"
	evidence "github.com/cosmos/cosmos-sdk/x/evidence/types"
	feegrant "github.com/cosmos/cosmos-sdk/x/feegrant"
	gov "github.com/cosmos/cosmos-sdk/x/gov/types"
	proposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	slashing "github.com/cosmos/cosmos-sdk/x/slashing/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking/types"
	upgrade "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/dezswap/cosmwasm-etl/configs"
	grpcConn "github.com/dezswap/cosmwasm-etl/pkg/grpc"
	"github.com/dezswap/cosmwasm-etl/pkg/s3client"
)

type DataStore interface {
	// testing purpose
	SetNewServiceClientFunc(func(cc grpc1.ClientConn) txtypes.ServiceClient)
	SetNewS3ClientFunc(func() (s3client.S3ClientInterface, error))
	SetNewQueryClientFunc(func(cc grpc1.ClientConn) wasm.QueryClient)

	// additional wrapper for Collector module usage
	GetNodeSyncedHeight() (int64, error)
	GetChainId() string
	GetLatestProcessedBlockNumber(...string) (int64, error)
	ChangeLatestBlock(int64, ...string) error
	UploadBlockBinary(int64, []byte, ...string) error

	// exposed to public usage in actual
	AddCustomInterfaceRegistry(...func(codectypes.InterfaceRegistry))
	GetBlockByHeight(int64) (*tendermintType.Block, error)
	GetBlockTxsFromHeight(int64) (*BlockTxsRaw, error)
	GetBlockTxsFromBlockData(*tendermintType.Block) (*BlockTxsRaw, error)
	GetCurrentPairsList(int64) (*PairListDTO, error)
	GetCurrentPoolStatusOfUnitPair(int64, string) (*PoolInfoDTO, error)
	GetCurrentPoolStatusOfAllPairs(int64) (*PoolInfoList, error)
	GetPoolStatusOfUnitPairByHeight(int64, string, ...string) (*PoolInfoDTO, error)
	GetPoolStatusOfAllPairsByHeight(int64, ...string) (*PoolInfoList, error)
	UploadPoolInfoBinary(int64, []byte, ...string) error
}

const (
	BLOCK_SUFFIX = "block"
	PAIR_SUFFIX  = "pair"
)

func GetBlockFolderPath(chainId string) []string {
	return []string{chainId, BLOCK_SUFFIX}
}

func GetPairFolderPath(chainId string) []string {
	return []string{chainId, PAIR_SUFFIX}
}

type dataStoreImpl struct {
	cdc       *codec.ProtoCodec
	legacyCdc *codec.LegacyAmino

	interfaceRegistry      codectypes.InterfaceRegistry
	chainId                string
	FactoryContractAddress string

	serviceDesc grpcConn.ServiceDesc
	s3Client    s3client.S3ClientInterface

	lcdClient LcdClient

	newServiceClientFunc func(cc grpc1.ClientConn) txtypes.ServiceClient
	newQueryClientFunc   func(cc grpc1.ClientConn) wasm.QueryClient
	newS3ClientFunc      func() (s3client.S3ClientInterface, error)
}

var _ DataStore = &dataStoreImpl{}

func New(c configs.Config, serviceDesc grpcConn.ServiceDesc, lcd LcdClient) (DataStore, error) {
	dataStoreService := &dataStoreImpl{
		serviceDesc: serviceDesc,
		lcdClient:   lcd,
	}

	dataStoreService.newServiceClientFunc = txtypes.NewServiceClient
	dataStoreService.newS3ClientFunc = s3client.NewClient
	dataStoreService.newQueryClientFunc = wasm.NewQueryClient
	dataStoreService.chainId = c.Collector.ChainId
	dataStoreService.FactoryContractAddress = c.Collector.PairFactoryContractAddress

	dataStoreService.initInterfaceRegistry()

	return dataStoreService, nil
}

func (store *dataStoreImpl) GetChainId() string {
	return store.chainId
}

func (store *dataStoreImpl) GetNodeSyncedHeight() (int64, error) {
	conn := store.serviceDesc.GetConnection()
	client := tmservice.NewServiceClient(conn)
	res, err := client.GetLatestBlock(context.Background(), &tmservice.GetLatestBlockRequest{})
	if err != nil {
		return 0, errors.Wrap(err, "dataStoreImpl.GetNodeSyncedHeight")
	}

	return res.Block.Header.Height, nil
}

// get latest processed block number
// you may crawl from the next number of this return
func (store *dataStoreImpl) GetLatestProcessedBlockNumber(folderPath ...string) (int64, error) {
	var err error
	store.s3Client, err = store.newS3ClientFunc()
	if err != nil {
		err = errors.Wrap(err, "GetLatestProcessedBlockNumber, S3 client create")
		return -1, err
	}

	latestBlock, err := store.s3Client.GetLatestProcessedBlockNumber(folderPath...)
	if err != nil {
		return -1, err
	}

	return latestBlock, nil
}

// get block data by height
// the data comes from the node, not S3
func (store *dataStoreImpl) GetBlockByHeight(height int64) (*tendermintType.Block, error) {
	conn := store.serviceDesc.GetConnection()
	client := store.newServiceClientFunc(conn)

	req := &txtypes.GetBlockWithTxsRequest{
		Height: height,
	}

	resp, err := client.GetBlockWithTxs(context.Background(), req)

	// TODO: delete this line after cosmos-sdk 0.46 is applied
	if err != nil && strings.Contains(err.Error(), "cannot paginate 0 txs with offset 0 and limit 100") {
		block := &tendermintType.Block{
			Header: tendermintType.Header{
				Height:  height,
				ChainID: store.GetChainId(),
			},

			Data: tendermintType.Data{
				Txs: [][]byte{},
			},
		}

		return block, nil
	} else if err != nil {
		err = errors.Wrap(err, "GetBlockByHeight, GetBlockWithTxs")
		//TODO FAILOVER
		return nil, err
	}

	return resp.GetBlock(), nil
}

// get tx data from the given block
// the data comes from the node, not S3
func (store *dataStoreImpl) GetBlockTxsFromHeight(height int64) (*BlockTxsRaw, error) {
	block, err := store.GetBlockByHeight(height)
	if err != nil {
		err = errors.Wrap(err, "GetBlockTxsFromHeight, GetBlockByHeight")
		return nil, err
	}

	info, err := store.GetBlockTxsFromBlockData(block)
	if err != nil {
		err = errors.Wrap(err, "GetBlockTxsFromHeight, GetBlockTxsFromBlockData")
		return nil, err
	}

	return info, nil
}

// get tx data from the given block data
// the data comes from the node, not S3
func (store *dataStoreImpl) GetBlockTxsFromBlockData(block *tendermintType.Block) (*BlockTxsRaw, error) {
	blockTxs, err := store.extractTxMsgs(block)
	if err != nil {
		err = errors.Wrap(err, "GetBlockTxsFromBlockData, extractTxMsgs")
		return nil, err
	}

	info := BlockTxsRaw{
		BlockId: block.GetHeader().Height,
		Txs:     blockTxs,
	}

	return &info, nil
}

func (store *dataStoreImpl) GetCurrentPairsList(height int64) (*PairListDTO, error) {
	conn := store.serviceDesc.GetConnection()

	ret, err := store.getCurrentPairsList(conn, height)
	if err != nil {
		err = errors.Wrap(err, "GetCurrentPairsList, getCurrentPairsList")
		return nil, err
	}

	return ret, nil
}

func (store *dataStoreImpl) GetCurrentPoolStatusOfUnitPair(height int64, pairContract string) (*PoolInfoDTO, error) {
	conn := store.serviceDesc.GetConnection()

	ret, err := store.getCurrentPoolStatusOfUnitPair(conn, height, pairContract)
	if err != nil {
		err = errors.Wrap(err, "GetCurrentPoolStatusOfUnitPair, getCurrentPoolStatusOfUnitPair")
		return nil, err
	}

	return ret, nil
}

func (store *dataStoreImpl) GetCurrentPoolStatusOfAllPairs(height int64) (*PoolInfoList, error) {
	conn := store.serviceDesc.GetConnection()

	pairLists, err := store.getCurrentPairsList(conn, height)
	if err != nil {
		err = errors.Wrap(err, "GetCurrentPoolStatusOfAllPairs, getCurrentPairsList")
		return nil, err
	}

	ret := &PoolInfoList{Pairs: make(map[string]PoolInfoDTO)}

	for _, val := range pairLists.Pairs {
		unitPairInfo, err := store.getCurrentPoolStatusOfUnitPair(conn, height, val.ContractAddr)
		if err != nil {
			err = errors.Wrap(err, "GetCurrentPoolStatusOfAllPairs, getCurrentPoolStatusOfUnitPair")
			return nil, err
		}

		ret.Pairs[val.ContractAddr] = *unitPairInfo
	}

	return ret, nil
}

// From S3
func (store *dataStoreImpl) GetPoolStatusOfAllPairsByHeight(height int64, folderPath ...string) (*PoolInfoList, error) {
	var err error
	store.s3Client, err = store.newS3ClientFunc()
	if err != nil {
		err = errors.Wrap(err, "GetPoolStatusOfAllPairsByHeight, S3 client create")
		return nil, err
	}

	fileName := append(folderPath, fmt.Sprintf("%d.json", height))

	byteData, err := store.s3Client.GetFileFromS3(fileName...)
	if err != nil {
		err = errors.Wrap(err, "GetPoolStatusOfAllPairsByHeight, GetFileFromS3")
		return nil, err
	}

	ret := &PoolInfoList{}
	err = json.Unmarshal(byteData, ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

// From S3
func (store *dataStoreImpl) GetPoolStatusOfUnitPairByHeight(height int64, pairContract string, folderPath ...string) (*PoolInfoDTO, error) {
	pairList, err := store.GetPoolStatusOfAllPairsByHeight(height, folderPath...)
	if err != nil {
		err = errors.Wrap(err, "GetPoolStatusOfUnitPairByHeight, GetPoolStatusOfAllPairsByHeight")
		return nil, err
	}

	if ret, ok := pairList.Pairs[pairContract]; ok {
		return &ret, nil
	} else {
		return nil, fmt.Errorf("no pair %s info in the height %d", pairContract, height)
	}
}

func (store *dataStoreImpl) ChangeLatestBlock(height int64, folderPath ...string) error {
	if height == 0 {
		return nil
	}

	var err error
	store.s3Client, err = store.newS3ClientFunc()
	if err != nil {
		err = errors.Wrap(err, "GetLatestProcessedBlockNumber, S3 client create")
		return err
	}

	err = store.s3Client.ChangeLatestBlock(height, folderPath...)

	if err != nil {
		err = errors.Wrap(err, "UnmarkLatestBlock, UnmarkLatestBlock")
	}

	return err
}

func (store *dataStoreImpl) UploadBlockBinary(height int64, data []byte, folderPath ...string) error {
	var err error
	store.s3Client, err = store.newS3ClientFunc()
	if err != nil {
		err = errors.Wrap(err, "GetLatestProcessedBlockNumber, S3 client create")
		return err
	}

	err = store.s3Client.UploadBlockBinary(height, data, folderPath...)

	if err != nil {
		err = errors.Wrap(err, "UploadBlockBinaryAsLatest, UploadBlockBinaryAsLatest")
	}
	return err
}

func (store *dataStoreImpl) UploadPoolInfoBinary(height int64, data []byte, folderPath ...string) error {
	var err error
	store.s3Client, err = store.newS3ClientFunc()
	if err != nil {
		err = errors.Wrap(err, "GetLatestProcessedBlockNumber, S3 client create")
		return err
	}

	fileName := append(folderPath, fmt.Sprintf("%d.json", height))
	err = store.s3Client.UploadFileToS3(data, fileName...)

	if err != nil {
		err = errors.Wrap(err, "UploadPoolInfoBinary, UploadFileToS3")
	}

	return err
}

func (store *dataStoreImpl) initInterfaceRegistry() {
	store.interfaceRegistry, store.cdc = initProtoCodec()
	store.legacyCdc = initLegacyCodec()
}

// if other blockchain mainnet needs additional module, add all interface from custom module
func (store *dataStoreImpl) AddCustomInterfaceRegistry(customRegisterInterfaces ...func(codectypes.InterfaceRegistry)) {
	for _, unitCustomFunc := range customRegisterInterfaces {
		unitCustomFunc(store.interfaceRegistry)
	}

	store.cdc = codec.NewProtoCodec(store.interfaceRegistry)
}

func (store *dataStoreImpl) SetNewServiceClientFunc(clientFunc func(cc grpc1.ClientConn) txtypes.ServiceClient) {
	store.newServiceClientFunc = clientFunc
}

func (store *dataStoreImpl) SetNewS3ClientFunc(s3Func func() (s3client.S3ClientInterface, error)) {
	store.newS3ClientFunc = s3Func
}

func (store *dataStoreImpl) SetNewQueryClientFunc(queryFunc func(cc grpc1.ClientConn) wasm.QueryClient) {
	store.newQueryClientFunc = queryFunc
}

func (store *dataStoreImpl) getTxResultFromTxHash(txHash string) (*txtypes.GetTxResponse, error) {
	conn := store.serviceDesc.GetConnection()
	client := store.newServiceClientFunc(conn)

	req := &txtypes.GetTxRequest{
		Hash: txHash,
	}

	resp, err := client.GetTx(context.Background(), req)
	if err != nil {
		// failover with lcd
		if store.lcdClient != nil {
			resp, err = store.lcdClient.GetTx(txHash)
		}
		if err != nil {
			return nil, errors.Wrap(err, "getTxResultFromTxHash, GetTx")
		}
	}

	return resp, nil
}

func (store *dataStoreImpl) extractTxMsgs(block *tendermintType.Block) ([]TxRaw, error) {
	ret := []TxRaw{}
	rawTxs := block.Data.Txs

	for _, unitTx := range rawTxs {
		var err error

		txHash := store.getTxHash(unitTx)

		resp, err := store.getTxResultFromTxHash(txHash)
		if err != nil {
			err = errors.Wrap(err, "extractTxMsgs, getTxResultFromTxHash")
			return nil, err
		}

		unitTxDTO := TxRaw{
			TxHash:    txHash,
			TxContent: resp,
		}

		ret = append(ret, unitTxDTO)
	}

	return ret, nil
}

func (store *dataStoreImpl) getTxHash(rawTx []byte) string {
	hash := sha256.New()
	hash.Write(rawTx)

	md := hash.Sum(nil)

	return strings.ToUpper(hex.EncodeToString(md))
}

func (store *dataStoreImpl) getCurrentPairsList(conn *grpc.ClientConn, height int64) (*PairListDTO, error) {
	type pairsReq struct {
		Pairs struct {
			StartAfter *[2]AssetInfo `json:"start_after,omitempty"`
		} `json:"pairs"`
	}
	req := pairsReq{}
	ret := &PairListDTO{
		Pairs: make(map[string]UnitPairDTO),
	}
	for {
		reqMsg, err := json.Marshal(&req)
		if err != nil {
			return nil, errors.Wrap(err, "datastoreImpl.getCurrentPairsList")
		}

		queryByte, err := store.contractQuery(conn, height, store.FactoryContractAddress, string(reqMsg))
		if err != nil {
			err = errors.Wrap(err, "getCurrentPairsList, contractQuery")
			return nil, err
		}

		rawRet := &PairListResponse{}
		err = json.Unmarshal(queryByte, rawRet)
		if err != nil {
			err = errors.Wrap(err, "getCurrentPairsList, Unmarshal")
			return nil, err
		}
		if len(rawRet.Pairs) == 0 {
			break
		}

		for _, unit := range rawRet.Pairs {
			ret.Pairs[unit.ContractAddr] = *unit.Convert()
		}
		req.Pairs.StartAfter = &rawRet.Pairs[len(rawRet.Pairs)-1].AssetInfo
	}
	return ret, nil
}

func (store *dataStoreImpl) getCurrentPoolStatusOfUnitPair(conn *grpc.ClientConn, height int64, pairContract string) (*PoolInfoDTO, error) {
	queryMsg := `{"pool": {}}`

	queryByte, err := store.contractQuery(conn, height, pairContract, queryMsg)
	if err != nil {
		err = errors.Wrap(err, "getCurrentPoolStatusOfUnitPair, contractQuery")
		return nil, err
	}

	rawRet := &PoolInfo{}
	err = json.Unmarshal(queryByte, rawRet)
	if err != nil {
		err = errors.Wrap(err, "getCurrentPoolStatusOfUnitPair, Unmarshal")
		return nil, err
	}

	return rawRet.Convert(), nil
}

func (store *dataStoreImpl) contractQuery(conn *grpc.ClientConn, height int64, contractAddress string, msg string) ([]byte, error) {
	client := store.newQueryClientFunc(conn)

	ctx := metadata.AppendToOutgoingContext(context.Background(), grpctypes.GRPCBlockHeightHeader, strconv.FormatInt(height, 10))
	queryResp, err := client.SmartContractState(ctx, &wasm.QuerySmartContractStateRequest{
		Address:   contractAddress,
		QueryData: []byte(msg),
	})

	if err != nil && strings.Contains(err.Error(), "contract: not found") {
		// return empty pair list, and will no query to pool info
		return []byte(`{"pairs":[]}`), nil
	} else if err != nil {
		err = errors.Wrap(err, "contractQuery, SmartContractState")
		return nil, err
	}

	queryByte := queryResp.Data.Bytes()

	return queryByte, nil
}

// Register all interface of Cosmos SDK as much as it can
func initProtoCodec() (codectypes.InterfaceRegistry, *codec.ProtoCodec) {
	// Proto codec
	newInterfaceRegistry := codectypes.NewInterfaceRegistry()

	cosmossdk.RegisterInterfaces(newInterfaceRegistry)
	txtypes.RegisterInterfaces(newInterfaceRegistry)
	auth.RegisterInterfaces(newInterfaceRegistry)
	authz.RegisterInterfaces(newInterfaceRegistry)
	bank.RegisterInterfaces(newInterfaceRegistry)
	crisis.RegisterInterfaces(newInterfaceRegistry)
	distribution.RegisterInterfaces(newInterfaceRegistry)
	evidence.RegisterInterfaces(newInterfaceRegistry)
	feegrant.RegisterInterfaces(newInterfaceRegistry)
	gov.RegisterInterfaces(newInterfaceRegistry)
	proposal.RegisterInterfaces(newInterfaceRegistry)
	slashing.RegisterInterfaces(newInterfaceRegistry)
	staking.RegisterInterfaces(newInterfaceRegistry)
	upgrade.RegisterInterfaces(newInterfaceRegistry)
	wasm.RegisterInterfaces(newInterfaceRegistry)
	cryptocodec.RegisterInterfaces(newInterfaceRegistry)

	cdc := codec.NewProtoCodec(newInterfaceRegistry)

	return newInterfaceRegistry, cdc
}

func initLegacyCodec() *codec.LegacyAmino {
	// Legacy amino codec
	legacyCdc := codec.NewLegacyAmino()
	cosmossdk.RegisterLegacyAminoCodec(legacyCdc)
	// txtypes.RegisterLegacyAminoCodec(store.legacyCdc)
	auth.RegisterLegacyAminoCodec(legacyCdc)
	// authz.RegisterLegacyAminoCodec(store.legacyCdc)
	bank.RegisterLegacyAminoCodec(legacyCdc)
	crisis.RegisterLegacyAminoCodec(legacyCdc)
	distribution.RegisterLegacyAminoCodec(legacyCdc)
	evidence.RegisterLegacyAminoCodec(legacyCdc)
	// feegrant.RegisterLegacyAminoCodec(store.legacyCdc)
	gov.RegisterLegacyAminoCodec(legacyCdc)
	proposal.RegisterLegacyAminoCodec(legacyCdc)
	slashing.RegisterLegacyAminoCodec(legacyCdc)
	staking.RegisterLegacyAminoCodec(legacyCdc)
	upgrade.RegisterLegacyAminoCodec(legacyCdc)
	wasm.RegisterLegacyAminoCodec(legacyCdc)
	// cryptocodec.RegisterLegacyAminoCodec(store.legacyCdc)

	return legacyCdc
}
