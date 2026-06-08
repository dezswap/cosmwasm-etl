package repo

import (
	"encoding/json"
	"fmt"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/db"
	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/pkg/errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type repoImpl struct {
	mapper
	db      *gorm.DB
	chainId string
}

func New(chainId string, dbConfig configs.RdbConfig) dex.Repo {
	gormDB, err := db.OpenGormPostgres(dbConfig)
	if err != nil {
		panic(err)
	}

	return &repoImpl{
		mapper:  &parserMapperImpl{},
		db:      gormDB,
		chainId: chainId,
	}
}

// GetSyncedHeight implements p_dex.Repo
func (r *repoImpl) GetSyncedHeight() (uint64, error) {
	syncedHeight := schemas.SyncedHeight{}
	tx := r.db.FirstOrCreate(&syncedHeight, schemas.SyncedHeight{ChainId: r.chainId})

	if tx.Error != nil {
		return 0, errors.Wrap(tx.Error, "repo.GetSyncedHeight")
	}
	return syncedHeight.Height, nil
}

// GetPairs implements p_dex.Repo
func (r *repoImpl) GetPairs() (map[string]dex.Pair, error) {
	pairs := []schemas.Pair{}
	tx := r.db.Where("chain_id = ?", r.chainId).Find(&pairs)
	if tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.GetPairs")
	}
	result := make(map[string]dex.Pair)
	for _, pair := range pairs {
		result[pair.Contract] = r.toPairDto(pair)
	}
	return result, nil
}

// Insert implements p_dex.Repo
func (r *repoImpl) Insert(srcHeight uint64, targetHeight uint64, txs []dex.ParsedTx, arg ...interface{}) error {
	if len(arg) != 2 {
		errMsg := fmt.Sprintf("invalid others(%v)", arg)
		return errors.New(errMsg)
	}

	pools, ok := arg[0].([]dex.PoolInfo)
	if !ok {
		errMsg := fmt.Sprintf("invalid pools(%v)", arg[0])
		return errors.New(errMsg)
	}
	pairs, ok := arg[1].([]dex.Pair)
	if !ok {
		errMsg := fmt.Sprintf("invalid pair(%v)", arg[1])
		return errors.New(errMsg)
	}

	parsedTxs := []schemas.ParsedTx{}
	for _, tx := range txs {
		parsedTxs = append(parsedTxs, r.toParsedTxModel(r.chainId, targetHeight, tx))
	}
	poolInfoTxs := []schemas.PoolInfo{}
	for _, pool := range pools {
		poolInfoTxs = append(poolInfoTxs, r.toPoolInfoModel(r.chainId, targetHeight, pool))
	}
	pairTxs := []schemas.Pair{}
	for _, pair := range pairs {
		pairTxs = append(pairTxs, r.toPairModel(r.chainId, pair))
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(schemas.Pair{}).CreateInBatches(pairTxs, len(pairTxs)).Error; err != nil {
			return errors.Wrap(err, "repo.Insert.Pair")
		}
		if err := tx.Model(schemas.ParsedTx{}).Omit("Id").CreateInBatches(parsedTxs, len(parsedTxs)).Error; err != nil {
			return errors.Wrap(err, "repo.Insert.ParsedTx")
		}
		if err := tx.Model(schemas.PoolInfo{}).CreateInBatches(poolInfoTxs, len(poolInfoTxs)).Error; err != nil {
			return errors.Wrap(err, "repo.Insert.PoolInfo")
		}
		if err := tx.Model(&schemas.SyncedHeight{}).Where("chain_id = ? AND height = ?", r.chainId, srcHeight).Update("height", targetHeight).Error; err != nil {
			return errors.Wrap(err, "repo.Insert.SyncedHeight")
		}
		return nil
	})
}

func (r *repoImpl) InsertPairValidationException(chainID string, contractAddress string) error {
	if err := r.db.Model(&schemas.PairValidationException{}).Create(schemas.PairValidationException{
		ChainId:  chainID,
		Contract: contractAddress,
	}).Error; err != nil {
		return errors.Wrap(err, "repo.InsertPairValidationException")
	}

	return nil
}

// ParsedPoolInfo implements p_dex.Repo.
func (r *repoImpl) ParsedPoolsInfo(from, to uint64) ([]dex.PoolInfo, error) {
	type poolInfo struct {
		Contract      string
		Asset0        string
		Asset0_amount string
		Asset1        string
		Asset1_amount string
		LpAmount      string
	}

	pools := []poolInfo{}
	if err := r.db.Model(&schemas.ParsedTx{}).Where(
		"chain_id = ? AND height >= ? AND height <= ?", r.chainId, from, to,
	).Select(
		"contract, MAX(asset0) as asset0, MAX(asset1) asset1, SUM(asset0_amount) as asset0_amount, SUM(asset1_amount) as asset1_amount, SUM(lp_amount) as lp_amount",
	).Group("contract").Scan(&pools).Error; err != nil {
		return nil, errors.Wrap(err, "repoImpl.ParsedPoolInfo")
	}

	results := []dex.PoolInfo{}
	for _, pool := range pools {
		results = append(results, dex.PoolInfo{
			ContractAddr: pool.Contract,
			Assets: []dex.Asset{
				{Addr: pool.Asset0, Amount: pool.Asset0_amount},
				{Addr: pool.Asset1, Amount: pool.Asset1_amount},
			},
			TotalShare: pool.LpAmount,
		})
	}

	return results, nil
}

// GetTokenExceptions implements dex.PairRepo.
func (r *repoImpl) GetTokenExceptions() (map[string]bool, error) {
	var rows []schemas.TokenParseException
	result := r.db.Where("chain_id = ?", r.chainId).Find(&rows)
	if result.Error != nil {
		return nil, errors.Wrap(result.Error, "repo.GetTokenExceptions")
	}
	m := make(map[string]bool, len(rows))
	for _, row := range rows {
		m[row.Contract] = true
	}
	return m, nil
}

// ValidationExceptionList implements dex.Repo.
func (r *repoImpl) ValidationExceptionList() ([]string, error) {
	exceptions := []string{}
	tx := r.db.Model(&schemas.PairValidationException{}).Where("chain_id = ?", r.chainId).Select("contract").Scan(&exceptions)
	if tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repoImpl.ValidationExceptionList")
	}
	return exceptions, nil
}

// GetValidationHeight implements dex.Repo.
func (r *repoImpl) GetValidationHeight() (uint64, error) {
	sh := schemas.SyncedHeight{}
	if tx := r.db.FirstOrCreate(&sh, schemas.SyncedHeight{ChainId: r.chainId}); tx.Error != nil {
		return 0, fmt.Errorf("GetValidationHeight: %w", tx.Error)
	}
	if sh.ValidationHeight == nil {
		return 0, nil
	}
	return *sh.ValidationHeight, nil
}

// SetValidationHeight implements dex.Repo.
func (r *repoImpl) SetValidationHeight(height uint64) error {
	tx := r.db.Model(&schemas.SyncedHeight{}).
		Where("chain_id = ?", r.chainId).
		Update("validation_height", height)
	if tx.Error != nil {
		return fmt.Errorf("SetValidationHeight: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return fmt.Errorf("SetValidationHeight: no row found for chain_id %s", r.chainId)
	}
	return nil
}

// ClearValidationHeight implements dex.Repo.
func (r *repoImpl) ClearValidationHeight() error {
	tx := r.db.Model(&schemas.SyncedHeight{}).
		Where("chain_id = ?", r.chainId).
		Update("validation_height", nil)
	if tx.Error != nil {
		return fmt.Errorf("ClearValidationHeight: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return fmt.Errorf("ClearValidationHeight: no row found for chain_id %s", r.chainId)
	}
	return nil
}

func (r *repoImpl) UpsertParseQuarantine(quarantine dex.ParseQuarantine) error {
	rawTx, err := json.Marshal(quarantine.RawTx)
	if err != nil {
		return errors.Wrap(err, "repo.UpsertParseQuarantine")
	}

	row := schemas.ParseQuarantine{
		ChainId:  r.chainId,
		Height:   quarantine.Height,
		Hash:     quarantine.Hash,
		Stage:    quarantine.Stage,
		Contract: quarantine.Contract,
		Action:   quarantine.Action,
		Error:    quarantine.Error,
		RawTx:    schemas.JSON(rawTx),
		Status:   dex.QuarantineStatusPending,
	}
	tx := r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "chain_id"}, {Name: "hash"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"height":      row.Height,
			"stage":       row.Stage,
			"contract":    row.Contract,
			"action":      row.Action,
			"error":       row.Error,
			"raw_tx":      row.RawTx,
			"status":      row.Status,
			"resolved_at": nil,
			"updated_at":  gorm.Expr("EXTRACT(EPOCH FROM NOW())"),
		}),
	}).Create(&row)
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "repo.UpsertParseQuarantine")
	}
	return nil
}

func (r *repoImpl) PendingParseQuarantines() ([]dex.ParseQuarantine, error) {
	var rows []schemas.ParseQuarantine
	tx := r.db.Where("chain_id = ? AND status = ?", r.chainId, dex.QuarantineStatusPending).
		Order("height ASC, id ASC").
		Find(&rows)
	if tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.PendingParseQuarantines")
	}

	result := make([]dex.ParseQuarantine, 0, len(rows))
	for _, row := range rows {
		var rawTx parser.RawTx
		if err := json.Unmarshal(row.RawTx, &rawTx); err != nil {
			return nil, errors.Wrapf(err, "repo.PendingParseQuarantines id=%d", row.Id)
		}
		result = append(result, dex.ParseQuarantine{
			ID:       row.Id,
			Height:   row.Height,
			Hash:     row.Hash,
			Stage:    row.Stage,
			Contract: row.Contract,
			Action:   row.Action,
			Error:    row.Error,
			RawTx:    rawTx,
		})
	}
	return result, nil
}

func (r *repoImpl) ResolveParseQuarantine(id uint64, height uint64, txs []dex.ParsedTx) error {
	parsedTxs := make([]schemas.ParsedTx, 0, len(txs))
	for _, tx := range txs {
		parsedTxs = append(parsedTxs, r.toParsedTxModel(r.chainId, height, tx))
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		// The whole transaction is persisted here to avoid partial, non-idempotent replay results.
		if len(parsedTxs) > 0 {
			if err := tx.Model(schemas.ParsedTx{}).Omit("Id").CreateInBatches(parsedTxs, len(parsedTxs)).Error; err != nil {
				return errors.Wrap(err, "repo.ResolveParseQuarantine.ParsedTx")
			}
		}
		result := tx.Model(&schemas.ParseQuarantine{}).
			Where("id = ? AND chain_id = ? AND status = ?", id, r.chainId, dex.QuarantineStatusPending).
			Updates(map[string]interface{}{
				"status":      dex.QuarantineStatusResolved,
				"resolved_at": gorm.Expr("EXTRACT(EPOCH FROM NOW())"),
				"updated_at":  gorm.Expr("EXTRACT(EPOCH FROM NOW())"),
			})
		if result.Error != nil {
			return errors.Wrap(result.Error, "repo.ResolveParseQuarantine.Quarantine")
		}
		if result.RowsAffected != 1 {
			return errors.Errorf("repo.ResolveParseQuarantine: pending quarantine not found id=%d", id)
		}
		return nil
	})
}
