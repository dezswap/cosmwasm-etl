package fcd

import (
	"strconv"
	"strings"
	"time"

	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/fcd"
	"github.com/pkg/errors"
)

type columbus4FcdRepo struct {
	client fcd.Fcd
	col4Mapper
}

func NewFcdRepo(client fcd.Fcd) fcdRepo {
	return &columbus4FcdRepo{client, &col4RepoMapper{}}
}

// HasMoreTx implements fcdRepo.
func (rp *columbus4FcdRepo) HasMoreTx(addr string, collectedOffset uint32) (bool, error) {
	limit := fcd.FCD_TXS_MAX_LIMIT
	offset := int(collectedOffset)
	txsRes, err := rp.client.TxsOf(addr, fcd.FcdTxsReqQuery{Limit: &limit, Offset: &offset})
	if err != nil {
		return false, errors.Wrap(err, "columbus4SrcRepo.TxsOf")
	}
	return len(txsRes.Txs) != 0, nil
}

// / TxsOf retrieves transactions from the FCD server and assumes that transactions greater than the offset have already been stored.
//
// @collectedOffset the offset of the last transaction collected from the FCD server.
//
// @untilHeight desired block until which transactions are intended to be collected.
// it is used to find the starting offset (the FCD server returns in reverse order of height)
func (rp *columbus4FcdRepo) TxsOf(addr string, collectedOffset, untilHeight, maxLen uint32) ([]schemas.FcdTxLog, error) {
	startOffset := collectedOffset
	var err error
	if collectedOffset == 0 {
		startOffset, err = rp.findStartOffset(addr, untilHeight)
		if err != nil {
			return nil, errors.Wrap(err, "columbus4SrcRepo.TxsOf")
		}
	}

	errLimit := 3
	limit := fcd.FCD_TXS_MAX_LIMIT
	offset := int(startOffset)
	txs := []schemas.FcdTxLog{}
	for ; ; offset = int(txs[len(txs)-1].FcdOffset) {
		txsRes, err := rp.client.TxsOf(addr, fcd.FcdTxsReqQuery{Limit: &limit, Offset: &offset})
		if err != nil {
			if strings.Contains(err.Error(), fcd.STATUS_SERVER_ERROR.Error()) && 0 < errLimit {
				errLimit--
				time.Sleep(10 * time.Second)
				continue
			}
			return nil, errors.Wrap(err, "columbus4SrcRepo.TxsOf")
		}
		if len(txsRes.Txs) == 0 {
			break
		}

		tmp, err := rp.toTxs(addr, txsRes.Txs)
		if err != nil {
			return nil, errors.Wrap(err, "columbus4SrcRepo.TxsOf")
		}
		txs = append(txs, tmp...)
		if len(txs) >= int(maxLen) {
			return txs, nil
		}
	}

	return txs, nil
}

func (rp *columbus4FcdRepo) findStartOffset(addr string, targetHeight uint32) (uint32, error) {

	limit := fcd.FCD_TXS_MIN_LIMIT
	l, r := 0, fcd.FCD_TXS_MAX_OFFSET
	mid := (l + r) / 2

	for ; l < r; mid = (l + r) / 2 {
		txsRes, err := rp.client.TxsOf(addr, fcd.FcdTxsReqQuery{Limit: &limit, Offset: &mid})
		if err != nil {
			return 0, errors.Wrap(err, "columbus4SrcRepo.TxsOf")
		}

		if len(txsRes.Txs) == 0 {
			l = mid + 1
			continue
		}

		height, err := strconv.ParseUint(txsRes.Txs[0].Height, 10, 64)
		if err != nil {
			return 0, errors.Wrap(err, "columbus4SrcRepo.TxsOf")
		}

		if targetHeight < uint32(height) {
			r = mid
		} else {
			l = mid + 1
		}
	}
	return uint32(l - 1), nil
}

// toTxs implements col4Mapper.
func (m *col4RepoMapper) toTxs(addr string, txsRes []fcd.FcdTxRes) ([]schemas.FcdTxLog, error) {
	txs := make([]schemas.FcdTxLog, len(txsRes))
	for i, txRes := range txsRes {
		height, err := strconv.ParseUint(txRes.Height, 10, 64)
		if err != nil {
			return nil, errors.Wrap(err, "col4RepoMapper.toTxs")
		}
		ts, err := time.Parse(time.RFC3339, txRes.Timestamp)
		if err != nil {
			return nil, errors.Wrap(err, "col4RepoMapper.toTxs")
		}
		txs[i] = schemas.FcdTxLog{
			FcdOffset: uint32(txRes.Id),
			Height:    uint32(height),
			Timestamp: ts,
			Address:   addr,
			EventLog:  txRes.RawLog,
			Hash:      txRes.TxHash,
		}
	}

	return txs, nil
}

type col4Mapper interface {
	toTxs(addr string, txs []fcd.FcdTxRes) (Txs, error)
}
type col4RepoMapper struct{}

var _ col4Mapper = (*col4RepoMapper)(nil)
var _ fcdRepo = (*columbus4FcdRepo)(nil)
