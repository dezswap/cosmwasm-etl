package datastore

import (
	"encoding/json"
	"fmt"

	"github.com/dezswap/cosmwasm-etl/pkg/s3client"
	"github.com/pkg/errors"
)

type ReadStore interface {
	GetLatestHeight() (uint64, error)
	GetBlockByHeight(height uint64) (*BlockTxsDTO, error)
	GetPoolStatusOfAllPairsByHeight(uint64) (*PoolInfoList, error)
}

type readStore struct {
	chainId  string
	s3Client s3client.S3ClientInterface
}

var _ ReadStore = &readStore{}

func NewReadStore(chainId string, s3Client s3client.S3ClientInterface) ReadStore {
	return &readStore{chainId, s3Client}
}

// GetBlockByHeight implements ReadStore
func (r *readStore) GetBlockByHeight(height uint64) (*BlockTxsDTO, error) {
	fileKey := r.s3Client.GetBlockFilePath(int64(height), GetBlockFolderPath(r.chainId)...)

	data, err := r.s3Client.GetFileFromS3(fileKey...)
	if err != nil {
		return nil, errors.Wrap(err, "readStore.GetBlockByHeight")
	}
	block := BlockTxsDTO{}
	if err := json.Unmarshal(data, &block); err != nil {
		return nil, errors.Wrap(err, "readStore.GetBlockByHeight")
	}
	return &block, nil
}

// GetLatestHeight implements ReadStore
func (r *readStore) GetLatestHeight() (uint64, error) {
	height, err := r.s3Client.GetLatestProcessedBlockNumber(GetBlockFolderPath(r.chainId)...)
	if err != nil {
		return uint64(0), errors.Wrap(err, "readStore.GetLatestHeight")
	}
	if height < 0 {
		return uint64(0), errors.New("returned height is negative")
	}
	return uint64(height), nil
}

// GetPoolStatusOfAllPairsByHeight implements ReadStore
func (r *readStore) GetPoolStatusOfAllPairsByHeight(height uint64) (*PoolInfoList, error) {
	fileName := append(GetPairFolderPath(r.chainId), fmt.Sprintf("%d.json", height))
	byteData, err := r.s3Client.GetFileFromS3(fileName...)
	if err != nil {
		return nil, errors.Wrap(err, "readStore.GetPoolStatusOfAllPairsByHeight")
	}

	ret := &PoolInfoList{}
	err = json.Unmarshal(byteData, ret)
	if err != nil {
		return nil, errors.Wrap(err, "readStore.GetPoolStatusOfAllPairsByHeight")
	}

	return ret, nil
}
