package router

import (
	"log"
	"os"
	"time"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/pkg/db"
	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type SrcRepo interface {
	Pairs() ([]Pair, error)
	UpdateRoutes(indexToAsset map[int]string, routesMap map[int]map[int][][]int) error
}

var _ SrcRepo = &srcRepoImpl{}

type srcRepoImpl struct {
	db      *gorm.DB
	chainId string
}

func NewSrcRepo(chainId string, dbConfig configs.RdbConfig) SrcRepo {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,  // Slow SQL threshold
			LogLevel:                  logger.Error, // Log level
			IgnoreRecordNotFoundError: true,         // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,        // Disable color
		},
	)

	pq := db.PostgresDb{}
	err := pq.Init(dbConfig)
	if err != nil {
		panic(err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: pq.Db,
	}), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		panic(err)
	}

	return &srcRepoImpl{
		db:      gormDB,
		chainId: chainId,
	}
}

func (r *srcRepoImpl) Pairs() ([]Pair, error) {
	pairs := []schemas.Pair{}
	tx := r.db.Where(schemas.Pair{ChainId: r.chainId}).Find(&pairs)
	if tx.Error != nil {
		return nil, errors.Wrap(tx.Error, "repo.Pairs")
	}

	newPairs := []Pair{}
	for _, p := range pairs {
		assetInfos := []string{p.Asset0, p.Asset1}
		newPairs = append(newPairs, Pair{Contract: p.Contract, AssetInfos: assetInfos})
	}

	return newPairs, nil
}

func (r *srcRepoImpl) UpdateRoutes(indexToAsset map[int]string, routesMap map[int]map[int][][]int) error {
	dbRoutes := make([]schemas.Route, 0)
	for a0, a1Routes := range routesMap {
		for a1, routes := range a1Routes {
			for _, route := range routes {
				assetRoute := make(pq.StringArray, len(route))
				for i, a := range route {
					assetRoute[i] = indexToAsset[a]
				}

				dbRoutes = append(dbRoutes, schemas.Route{
					ChainId:  r.chainId,
					Asset0:   indexToAsset[a0],
					Asset1:   indexToAsset[a1],
					HopCount: len(route) - 1,
					Route:    assetRoute,
				})
			}
		}
	}

	tx := r.db.Model(schemas.Route{}).Clauses(
		clause.OnConflict{
			Columns:   []clause.Column{{Name: "chain_id"}, {Name: "asset0"}, {Name: "asset1"}, {Name: "route"}},
			DoNothing: true,
		}).Create(&dbRoutes)
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "repo.UpdateRoutes")
	}

	return nil
}
