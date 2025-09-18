package checkpoint

import (
	"fmt"
	"github.com/dezswap/cosmwasm-etl/parser/dex"
	"github.com/pkg/errors"
	"math/big"
	"time"
)

type asset struct {
	Addr   string
	Amount *big.Int
}

type Builder struct {
	repo dex.Repo
	ds   dex.SourceDataStore
}

func NewBuilder(repo dex.Repo, ds dex.SourceDataStore) *Builder {
	return &Builder{
		repo: repo,
		ds:   ds,
	}
}

func (b *Builder) Build(targetHeight uint64) error {
	dbHeight, err := b.validateAndGetDbHeight(targetHeight)
	if err != nil {
		return errors.Wrap(err, "failed to check heights")
	}

	txs, pools, pairs, err := b.generateCheckpointData(targetHeight)
	if err != nil {
		return errors.Wrap(err, "failed to generate checkpoint data")
	}

	if len(txs) == 0 {
		fmt.Printf("No changes detected between heights(%d - %d).\n", dbHeight, targetHeight)
	}

	// Save checkpoint data (transactions, pools, pairs) to database
	if err := b.repo.Insert(dbHeight, targetHeight, txs, pools, pairs); err != nil {
		return errors.Wrap(err, "failed to insert data")
	}

	return nil
}

func (b *Builder) validateAndGetDbHeight(targetHeight uint64) (uint64, error) {
	dbHeight, err := b.repo.GetSyncedHeight()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get database synced height")
	}
	sourceHeight, err := b.ds.GetSourceSyncedHeight()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get source node synced height")
	}

	if dbHeight > sourceHeight {
		return 0, errors.New("database height is ahead of source node height")
	}

	if dbHeight >= targetHeight {
		return 0, errors.New("checkpoint already exists for target height")
	}

	if targetHeight > sourceHeight {
		return 0, errors.Errorf("target height is beyond source node's current height(%d)", sourceHeight)
	}

	return dbHeight, nil
}

func (b *Builder) generateCheckpointData(targetHeight uint64) ([]dex.ParsedTx, []dex.PoolInfo, []dex.Pair, error) {
	srcPoolInfos, err := b.ds.GetPoolInfos(targetHeight)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to get source pool infos")
	}
	rdbPoolInfos, err := b.repo.ParsedPoolsInfo(0, targetHeight)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to get rdb pool infos")
	}

	rdbPoolInfoMap := make(map[string]dex.PoolInfo)
	for _, pi := range rdbPoolInfos {
		rdbPoolInfoMap[pi.ContractAddr] = pi
	}

	var txs []dex.ParsedTx
	var pools []dex.PoolInfo
	var pairs []dex.Pair

	for _, pi := range srcPoolInfos {
		if v, ok := rdbPoolInfoMap[pi.ContractAddr]; ok {
			tx, err := createDiffTx(pi, v)
			if err != nil {
				return nil, nil, nil, errors.Wrapf(err, "failed to create diff tx for contract %s", pi.ContractAddr)
			}
			if tx != nil {
				txs = append(txs, *tx)
			}
		} else {
			txs = append(txs, createNewPairTxs(pi)...)
			pools = append(pools, pi)
		}
	}

	for _, tx := range txs {
		if tx.Type == dex.CreatePair {
			pairs = append(pairs, dex.Pair{
				ContractAddr: tx.ContractAddr,
				Assets:       []string{tx.Assets[0].Addr, tx.Assets[1].Addr},
				LpAddr:       tx.LpAddr,
			})
		}
	}

	return txs, pools, pairs, nil
}

func createDiffTx(src, rdb dex.PoolInfo) (*dex.ParsedTx, error) {
	assetDiff, err := calculateAssetDiff(src.Assets, rdb.Assets)
	if err != nil {
		return nil, err
	}
	if len(assetDiff) != 2 {
		return nil, errors.New("asset slice must contain exactly 2 assets")
	}
	isAssetDiffZero := true
	isAssetDiffPositiveAll := true
	isAssetDiffNegativeAll := true
	for _, ad := range assetDiff {
		sign := ad.Amount.Sign()
		if sign < 0 {
			isAssetDiffZero = false
			isAssetDiffPositiveAll = false
		} else if sign > 0 {
			isAssetDiffZero = false
			isAssetDiffNegativeAll = false
		}
	}

	spis, err := dex.ToBigInt(src.TotalShare)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert source total share to big int")
	}
	rpis, err := dex.ToBigInt(rdb.TotalShare)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert rdb total share to big int")
	}
	totalShareDiff := spis.Sub(spis, rpis)
	isTotalShareZero := totalShareDiff.Sign() == 0

	// nothing changed
	if isAssetDiffZero && isTotalShareZero {
		return nil, nil
	}

	txType := dex.Swap
	{
		// uncommon case, possibly due to direct transfers without minting LP
		// e.g. withdraw -> transfer: total share changed, asset diff zero
		// e.g. transfer: total share not changed, asset diff positive
		if isAssetDiffZero {
			txType = dex.Provide
		}

		if isAssetDiffPositiveAll {
			txType = dex.Provide
		}

		if isAssetDiffNegativeAll {
			txType = dex.Withdraw
		}
	}

	return &dex.ParsedTx{
		Hash:             "-",
		Timestamp:        time.Now(),
		Type:             txType,
		Sender:           "-",
		ContractAddr:     src.ContractAddr,
		Assets:           [2]dex.Asset(ToDexAssets(assetDiff)),
		LpAddr:           src.LpAddr,
		LpAmount:         totalShareDiff.String(),
		CommissionAmount: "0",
	}, nil
}

func calculateAssetDiff(src, rdb []dex.Asset) ([]asset, error) {
	var assetDiff []asset
	for _, sa := range src {
		for _, ra := range rdb {
			if sa.Addr != ra.Addr {
				continue
			}

			sbi, err := dex.ToBigInt(sa.Amount)
			if err != nil {
				return nil, errors.Wrap(err, "failed to convert source amount to big int")
			}
			rbi, err := dex.ToBigInt(ra.Amount)
			if err != nil {
				return nil, errors.Wrap(err, "failed to convert rdb amount to big int")
			}

			assetDiff = append(assetDiff, asset{sa.Addr, sbi.Sub(sbi, rbi)})
		}
	}
	return assetDiff, nil
}

func ToDexAssets(assets []asset) []dex.Asset {
	dexAssets := make([]dex.Asset, len(assets))

	for i, a := range assets {
		dexAssets[i] = dex.Asset{
			Addr:   a.Addr,
			Amount: a.Amount.String(),
		}
	}

	return dexAssets
}

func createNewPairTxs(pi dex.PoolInfo) []dex.ParsedTx {
	var emptyAssets [2]dex.Asset
	for i, a := range pi.Assets {
		emptyAssets[i] = dex.Asset{
			Addr: a.Addr, Amount: "0",
		}
	}

	return []dex.ParsedTx{
		{
			Hash:             "-",
			Timestamp:        time.Now(),
			Type:             dex.CreatePair,
			Sender:           "-",
			ContractAddr:     pi.ContractAddr,
			Assets:           emptyAssets,
			LpAddr:           pi.LpAddr,
			LpAmount:         "0",
			CommissionAmount: "0",
		},
		{
			Hash:             "-",
			Timestamp:        time.Now(),
			Type:             dex.InitialProvide,
			Sender:           "-",
			ContractAddr:     pi.ContractAddr,
			Assets:           [2]dex.Asset(pi.Assets),
			LpAddr:           pi.LpAddr,
			LpAmount:         pi.TotalShare,
			CommissionAmount: "0",
		},
	}
}
