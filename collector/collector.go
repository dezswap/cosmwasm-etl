package collector

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tendermintType "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/pkg/errors"

	"github.com/dezswap/cosmwasm-etl/collector/datastore"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
)

func DoCollect(store datastore.DataStore, logger logging.Logger) error {
	blockQueue := make(chan *tendermintType.Block, 10)
	errChan := make(chan error)

	blockFolderPath := datastore.GetBlockFolderPath(store.GetChainId())
	pairFolderPath := datastore.GetPairFolderPath(store.GetChainId())

	// block collecting job
	go func() {
		latestBlock, err := store.GetLatestProcessedBlockNumber(blockFolderPath...)
		if err != nil {
			err = errors.Wrap(err, "DoCollect, GetLatestProcessedBlockNumber")
			errChan <- err
			return
		}

		logger.Infof("Start collecting from the block %d ...", latestBlock+1)

		for {
			block, err := store.GetBlockByHeight(latestBlock + 1)
			if err != nil && strings.Contains(err.Error(), "invalid height") {
				// case 1: new block is not yet produced
				// TODO: print logger: "waiting new blocks..."

				timer := time.NewTimer(time.Second * 3)
				<-timer.C

				continue
			} else if err != nil {
				// case 2: other error
				err = errors.Wrap(err, "DoCollect, GetBlockByHeight")
				errChan <- err
				return
			}

			if block != nil {
				blockQueue <- block

				latestBlock = block.Header.Height
			} else {
				errChan <- fmt.Errorf("received block is nil")
				return
			}

			logger.Infof("Collected block %d ...", latestBlock)
		}
	}()

	// tx storing job
	for {
		select {
		case block := <-blockQueue:
			// store block
			blockId := block.GetHeader().Height

			logger.Infof("Processing block %d ...", blockId)

			blockInfo, err := store.GetBlockTxsFromBlockData(block)
			if err != nil {
				return errors.Wrap(err, "DoCollect, GetBlockTxsFromBlockData")
			}

			data, err := json.Marshal(blockInfo)
			if err != nil {
				return errors.Wrap(err, "DoCollect, Marshal")
			}

			err = store.UploadBlockBinary(blockId, data, blockFolderPath...)
			if err != nil {
				return errors.Wrap(err, "DoCollect, UploadBlockBinaryAsLatest")
			}

			logger.Infof("Pair processing in block %d ...", blockId)

			// store pair
			pairStatus, err := store.GetCurrentPoolStatusOfAllPairs(blockId)
			if err != nil {
				return errors.Wrap(err, "DoCollect, GetCurrentPoolStatusOfAllPairs")
			}

			pairStatusByte, err := json.Marshal(pairStatus)
			if err != nil {
				return errors.Wrap(err, "DoCollect, Marshal, pair")
			}

			err = store.UploadPoolInfoBinary(blockId, pairStatusByte, pairFolderPath...)
			if err != nil {
				return errors.Wrap(err, "DoCollect, UploadPoolInfoBinary, pair")
			}

			err = store.ChangeLatestBlock(blockId, blockFolderPath...)
			if err != nil {
				return errors.Wrap(err, "DoCollect, ChangeLatestBlock")
			}

			logger.Infof("Block %d process done!", blockId)

		case err := <-errChan:
			if err != nil {
				logger.Error(err)
				return err
			}
		}
	}
}
