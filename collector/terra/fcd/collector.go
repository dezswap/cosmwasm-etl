package fcd

import (
	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/pkg/errors"
)

type fcdRepo interface {
	TxsOf(addr string, collectedOffset, untilHeight, limit uint32) (Txs, error)
	HasMoreTx(addr string, collectedOffset uint32) (bool, error)
}

type permanentStore interface {
	FirstTxOf(addr string) (schemas.FcdTxLog, error)
	Inserts(txLogs []schemas.FcdTxLog) error
	TxLogsByHeight(height int) ([]schemas.FcdTxLog, error)
}

type ColumbusCollector interface {
	Collect(addr string, height uint32) error

	storeTxs(txs Txs) error
	hasCollected(addr string) (bool, error)

	fcdRepo
	permanentStore
}

type columbusFcdCollector struct {
	fcdRepo
	permanentStore
}

func New(repo fcdRepo, store permanentStore) ColumbusCollector {
	return &columbusFcdCollector{repo, store}
}

// hasCollected checks the absence of the previous transaction by examining the transactions collected from the FCD server
//
// NOTE: fcd returns transactions in reverse order of height
func (c *columbusFcdCollector) hasCollected(addr string) (bool, error) {
	tx, err := c.permanentStore.FirstTxOf(addr)
	if err != nil {
		return false, errors.Wrap(err, "col4permanentStore.HasCollected")
	}
	if tx.FcdOffset == 0 {
		return false, nil
	}

	hasMore, err := c.fcdRepo.HasMoreTx(addr, tx.FcdOffset)
	if err != nil {
		return false, errors.Wrap(err, "col4permanentStore.HasCollected")
	}
	return !hasMore, nil
}

// storeTxs implements permanentStore.
func (c *columbusFcdCollector) storeTxs(txs []schemas.FcdTxLog) error {
	err := c.Inserts(txs)
	if err != nil {
		return errors.Wrap(err, "col4permanentStore.StoreTxs")
	}

	return nil
}

// Collect implements ColumbusCollector.
func (c *columbusFcdCollector) Collect(address string, height uint32) error {
	for {
		collected, err := c.hasCollected(address)
		if err != nil {
			return errors.Wrap(err, "columbusFcdCollector.Collect")
		}
		if collected {
			return nil
		}

		tx, err := c.FirstTxOf(address)
		if err != nil {
			return errors.Wrap(err, "columbusFcdCollector.Collect")
		}

		// TODO: make configurable limit
		txs, err := c.TxsOf(address, tx.FcdOffset, height, 1000)
		if err != nil {
			return errors.Wrap(err, "columbusFcdCollector.Collect")
		}

		if err := c.storeTxs(txs); err != nil {
			return errors.Wrap(err, "columbusFcdCollector.Collect")
		}
	}
}
