package fcd

import (
	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type permanentStoreImpl struct {
	db *gorm.DB
}

var _ permanentStore = (*permanentStoreImpl)(nil)

func NewPermanentStore(db *gorm.DB) permanentStore {
	return &permanentStoreImpl{db}
}

// FirstTxOf implements Columbus4Repository.
func (p *permanentStoreImpl) FirstTxOf(addr string) (schemas.FcdTxLog, error) {
	tx := schemas.FcdTxLog{}
	if err := p.db.Where("address = ?", addr).Order("fcd_offset").Limit(1).Find(&tx).Error; err != nil {
		return schemas.FcdTxLog{}, errors.Wrap(err, "col4RepoImpl.FirstTxOf")
	}
	return tx, nil
}

// Inserts implements Columbus4Repository.
func (p *permanentStoreImpl) Inserts(txLogs []schemas.FcdTxLog) error {
	split := 1000
	for idx := 0; ; {
		end := idx + split
		if end > len(txLogs) {
			end = len(txLogs)
		}
		if err := p.db.CreateInBatches(txLogs[idx:end], len(txLogs[idx:end])).Error; err != nil {
			return errors.Wrap(err, "col4RepoImpl.Inserts")
		}
		if end == len(txLogs) {
			break
		}
		idx = end
	}
	return nil
}

// TxLogsByHeight implements Columbus4Repository.
func (p *permanentStoreImpl) TxLogsByHeight(height int) ([]schemas.FcdTxLog, error) {
	txs := []schemas.FcdTxLog{}
	if err := p.db.Where("height = ?", height).Find(&txs).Error; err != nil {
		return nil, errors.Wrap(err, "col4RepoImpl.TxLogsByHeight")
	}
	return txs, nil
}
