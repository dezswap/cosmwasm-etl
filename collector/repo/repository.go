package repo

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/parser"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/dezswap/cosmwasm-etl/pkg/db"
	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/lib/pq"
	pkgerrors "github.com/pkg/errors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNotFound    = errors.New("collector source data not found")
	ErrUnavailable = errors.New("collector source table unavailable")
)

// Repository stores parser-ready source data collected by DoCollectSource.
// Missing rows and missing tables are exposed separately so parser can fall
// back to direct chain reads without masking malformed stored data.
type Repository interface {
	GetSyncedHeight(chainID string) (uint64, error)
	GetBlockTxs(chainID string, height uint64) (parser.RawTxs, time.Time, error)
	GetPoolInfos(chainID string, height uint64) ([]dex.PoolInfo, error)
	SaveHeight(chainID string, height uint64, blockTime time.Time, txs parser.RawTxs, poolInfos []dex.PoolInfo, savePoolSnapshot bool) error
}

type repository struct {
	db *gorm.DB
}

var _ Repository = (*repository)(nil)

func New(dbConfig configs.RdbConfig) Repository {
	pqDB := db.PostgresDb{}
	if err := pqDB.Init(dbConfig); err != nil {
		panic(err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: pqDB.Db}), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	return NewWithDB(gormDB)
}

func NewWithDB(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetSyncedHeight(chainID string) (uint64, error) {
	row := schemas.CollectorSyncedHeight{}
	if err := r.db.Where("chain_id = ?", chainID).First(&row).Error; err != nil {
		return 0, classifyReadErr(err)
	}
	return row.Height, nil
}

func (r *repository) GetBlockTxs(chainID string, height uint64) (parser.RawTxs, time.Time, error) {
	row := schemas.CollectorBlock{}
	if err := r.db.Where("chain_id = ? AND height = ?", chainID, height).First(&row).Error; err != nil {
		return nil, time.Time{}, classifyReadErr(err)
	}

	txs := parser.RawTxs{}
	if err := json.Unmarshal(row.Txs, &txs); err != nil {
		return nil, time.Time{}, pkgerrors.Wrap(err, "collector repo unmarshal block txs")
	}
	return txs, row.BlockTime, nil
}

func (r *repository) GetPoolInfos(chainID string, height uint64) ([]dex.PoolInfo, error) {
	row := schemas.CollectorPoolSnapshot{}
	if err := r.db.Where("chain_id = ? AND height = ?", chainID, height).First(&row).Error; err != nil {
		return nil, classifyReadErr(err)
	}

	poolInfos := []dex.PoolInfo{}
	if err := json.Unmarshal(row.PoolInfos, &poolInfos); err != nil {
		return nil, pkgerrors.Wrap(err, "collector repo unmarshal pool infos")
	}
	return poolInfos, nil
}

func (r *repository) SaveHeight(chainID string, height uint64, blockTime time.Time, txs parser.RawTxs, poolInfos []dex.PoolInfo, savePoolSnapshot bool) error {
	txBytes, err := json.Marshal(txs)
	if err != nil {
		return pkgerrors.Wrap(err, "collector repo marshal block txs")
	}

	var poolBytes []byte
	if savePoolSnapshot {
		poolBytes, err = json.Marshal(poolInfos)
		if err != nil {
			return pkgerrors.Wrap(err, "collector repo marshal pool infos")
		}
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		block := schemas.CollectorBlock{
			ChainId:   chainID,
			Height:    height,
			BlockTime: blockTime.UTC(),
			Txs:       schemas.CollectorJSON(txBytes),
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := upsert(tx, block, []string{"chain_id", "height"}, []string{"block_time", "txs", "updated_at"}); err != nil {
			return pkgerrors.Wrap(err, "collector repo save block")
		}

		if savePoolSnapshot {
			pool := schemas.CollectorPoolSnapshot{
				ChainId:   chainID,
				Height:    height,
				PoolInfos: schemas.CollectorJSON(poolBytes),
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := upsert(tx, pool, []string{"chain_id", "height"}, []string{"pool_infos", "updated_at"}); err != nil {
				return pkgerrors.Wrap(err, "collector repo save pool snapshot")
			}
		}

		synced := schemas.CollectorSyncedHeight{
			ChainId:   chainID,
			Height:    height,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := upsert(tx, synced, []string{"chain_id"}, []string{"height", "updated_at"}); err != nil {
			return pkgerrors.Wrap(err, "collector repo save synced height")
		}
		return nil
	})
}

func upsert[T any](db *gorm.DB, value T, keyColumns []string, updateColumns []string) error {
	columns := make([]clause.Column, 0, len(keyColumns))
	for _, column := range keyColumns {
		columns = append(columns, clause.Column{Name: column})
	}
	return db.Clauses(clause.OnConflict{
		Columns:   columns,
		DoUpdates: clause.AssignmentColumns(updateColumns),
	}).Create(&value).Error
}

func classifyReadErr(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if isUndefinedTable(err) {
		return ErrUnavailable
	}
	return err
}

func isUndefinedTable(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "42P01" {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "sqlstate 42p01") ||
		(strings.Contains(msg, "relation") && strings.Contains(msg, "does not exist"))
}
