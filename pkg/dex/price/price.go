package price

import (
	"github.com/pkg/errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/dezswap/cosmwasm-etl/pkg/util"
)

type Price interface {
	CurrHeight() (int64, error)
	NextHeight(minHeight uint64) (int64, error)
	Run(height uint64) error
}

var _ Price = &priceImpl{}

type priceImpl struct {
	repo       SrcRepo
	priceToken string
	logger     logging.Logger
	mutex      *sync.Mutex

	tokenDecimals               map[string]int64
	priceRoutes                 map[string][][]string
	latestRouteUpdatedTimestamp time.Time
}

func New(repo SrcRepo, priceToken string, logger logging.Logger) (Price, error) {
	tokenDecimals := make(map[string]int64)
	priceTokenDecimal, err := repo.Decimals(priceToken)
	if err != nil {
		return nil, err
	}
	tokenDecimals[priceToken] = priceTokenDecimal

	p := &priceImpl{
		logger:        logger,
		priceToken:    priceToken,
		repo:          repo,
		mutex:         &sync.Mutex{},
		tokenDecimals: tokenDecimals,
	}

	return p, nil
}

func (p *priceImpl) CurrHeight() (int64, error) {
	height, err := p.repo.CurrHeight()
	if err != nil {
		return NaValue, err
	}

	return height, nil
}

func (p *priceImpl) NextHeight(minHeight uint64) (int64, error) {
	if minHeight == 0 {
		if firstHeight, err := p.repo.FirstHeight(p.priceToken); err != nil {
			return NaValue, err
		} else if firstHeight > 0 {
			minHeight = uint64(firstHeight) - 1
		}
	}
	height, err := p.repo.NextHeight(minHeight)
	if err != nil {
		return NaValue, err
	}

	return height, nil
}

func (p *priceImpl) Run(height uint64) error {
	if err := p.updatePriceRoute(); err != nil {
		return err
	}

	txs, err := p.repo.Txs(height)
	if err != nil {
		return err
	}

	for _, t := range txs {
		if t.Asset0Amount == "0" || t.Asset1Amount == "0" {
			continue
		}

		if t.Asset0 == p.priceToken || t.Asset1 == p.priceToken {
			if err := p.updateDirectSwapPrice(t); err != nil {
				return err
			}
		} else {
			if err := p.updateIndirectSwapPrice(t); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *priceImpl) updatePriceRoute() error {
	ts, err := p.repo.LatestRouteUpdateTimestamp()
	if err != nil {
		return err
	}
	lruts := util.ToTime(ts)
	if lruts.After(p.latestRouteUpdatedTimestamp) {
		p.priceRoutes, err = p.repo.Route(p.priceToken)
		if err != nil {
			return err
		}
		p.latestRouteUpdatedTimestamp = lruts
	}

	return nil
}

func (p *priceImpl) updateDirectSwapPrice(tx schemas.ParsedTx) error {
	isReverse := tx.Asset0 == p.priceToken

	var targetToken string
	var decimals0, decimals1 int64
	var err error

	if isReverse {
		targetToken = tx.Asset1
		decimals0 = p.tokenDecimals[p.priceToken]
		decimals1, err = p.decimals(tx.Asset1)
		if err != nil {
			return err
		}
	} else {
		targetToken = tx.Asset0
		decimals0, err = p.decimals(tx.Asset0)
		if err != nil {
			return err
		}
		decimals1 = p.tokenDecimals[p.priceToken]
	}

	price, err := p.calculatePrice(tx.Asset0Amount, decimals0, tx.Asset1Amount, decimals1, isReverse)
	if err != nil {
		return errors.Wrap(err, strings.Join([]string{
			"priceImpl.updateDirectSwapPrice: (Tx Hash: ", tx.Hash, ")"}, ""))
	}

	if err := p.repo.UpdateDirectPrice(tx.Height, tx.Id, targetToken, price.String(), p.priceToken, isReverse); err != nil {
		return err
	}

	return nil
}

func (p *priceImpl) calculatePrice(asset0Amount string, asset0Decimals int64, asset1Amount string, asset1Decimals int64, isReverse bool) (types.Dec, error) {
	asset0AmountD, err := util.StringAmountToDecimal(asset0Amount, asset0Decimals)
	if err != nil {
		return types.Dec{}, err
	}
	asset1AmountD, err := util.StringAmountToDecimal(asset1Amount, asset1Decimals)
	if err != nil {
		return types.Dec{}, err
	}

	if isReverse {
		return asset0AmountD.Quo(asset1AmountD).Abs(), nil
	}
	return asset1AmountD.Quo(asset0AmountD).Abs(), nil
}

func (p *priceImpl) decimals(token string) (int64, error) {
	var decimals int64
	var err error

	if d, ok := p.tokenDecimals[token]; ok {
		decimals = d
	} else {
		decimals, err = p.repo.Decimals(token)
		if err != nil {
			return NaValue, err
		}
		p.tokenDecimals[token] = decimals
	}

	return decimals, nil
}

func (p *priceImpl) updateIndirectSwapPrice(tx schemas.ParsedTx) error {
	decimals0, err := p.decimals(tx.Asset0)
	if err != nil {
		return errors.Wrap(err,
			strings.Join([]string{"priceImpl.updateIndirectSwapPrice: (Tx hash:", tx.Hash, ")"}, ""))
	}
	decimals1, err := p.decimals(tx.Asset1)
	if err != nil {
		return errors.Wrap(err,
			strings.Join([]string{"priceImpl.updateIndirectSwapPrice: (Tx hash:", tx.Hash, ")"}, ""))
	}

	route0, price0, liquidity0, err := p.optimalRoutePrice(tx.Height, tx.Asset0, decimals0)
	if err != nil {
		return errors.Wrap(err,
			strings.Join([]string{"priceImpl.updateIndirectSwapPrice: (Tx hash:", tx.Hash, ")"}, ""))
	}
	route1, price1, liquidity1, err := p.optimalRoutePrice(tx.Height, tx.Asset1, decimals1)
	if err != nil {
		return errors.Wrap(err,
			strings.Join([]string{"priceImpl.updateIndirectSwapPrice: (Tx hash:", tx.Hash, ")"}, ""))
	}

	if len(route0) == 0 && len(route1) == 0 {
		p.logger.Warnf("no price route found for a transaction(hash: %s)", tx.Hash)
		return nil
	}

	asset0AmountD, err := util.StringAmountToDecimal(tx.Asset0Amount, decimals0)
	if err != nil {
		return errors.Wrap(err,
			strings.Join([]string{"priceImpl.updateIndirectSwapPrice: (Tx hash:", tx.Hash, ")"}, ""))
	}
	asset1AmountD, err := util.StringAmountToDecimal(tx.Asset1Amount, decimals1)
	if err != nil {
		return errors.Wrap(err,
			strings.Join([]string{"priceImpl.updateIndirectSwapPrice: (Tx hash:", tx.Hash, ")"}, ""))
	}

	isPriceFixed := false
	/*
	 * use swap price when one of picked routes contains asset0 or asset1
	 * e.g. route0: asset0 -> asset1 -> price token, route1: asset1 -> price token
	 */
	if len(route0) > 1 && route0[0] == tx.Asset1 && len(route1) == len(route0)-1 {
		price0 = asset1AmountD.Quo(asset0AmountD).Mul(price1)
		isPriceFixed = true
	}
	if len(route1) > 1 && route1[0] == tx.Asset0 && len(route0) == len(route1)-1 {
		price1 = asset0AmountD.Quo(asset1AmountD).Mul(price0)
		isPriceFixed = true
	}

	/*
	 * pick one of price determined which liquidity holds more than the other
	 * e.g. route0: asset0 -> x -> y -> price token, route1: asset1 -> y -> price token
	 *      liquidity0: 10000, liquidity1: 20000
	 *      ==> fix route0 to asset0 -> asset1 -> y -> price token
	 */
	if !isPriceFixed {
		if liquidity0.GT(liquidity1) {
			containsAsset := false
			for _, a := range route0 {
				if a == tx.Asset1 {
					containsAsset = true
					break
				}
			}

			if !containsAsset {
				price1 = asset0AmountD.Quo(asset1AmountD).Mul(price0)
				route1 = append([]string{tx.Asset0}, route0...)
			}
		}

		if liquidity1.GT(liquidity0) {
			containsAsset := false
			for _, a := range route1 {
				if a == tx.Asset0 {
					containsAsset = true
					break
				}
			}

			if !containsAsset {
				price0 = asset1AmountD.Quo(asset0AmountD).Mul(price1)
				route0 = append([]string{tx.Asset1}, route1...)
			}
		}
	}

	if err := p.repo.UpdateRoutePrice(tx.Height, tx.Id, tx.Asset0, price0.Abs().String(), p.priceToken, route0); err != nil {
		return err
	}

	if err := p.repo.UpdateRoutePrice(tx.Height, tx.Id, tx.Asset1, price1.Abs().String(), p.priceToken, route1); err != nil {
		return err
	}

	return nil
}

func (p *priceImpl) optimalRoutePrice(height uint64, token string, decimals int64) ([]string, types.Dec, types.Dec, error) {
	var optimalRoute []string
	optimalPrice := types.ZeroDec()
	optimalRouteLiquidity := types.ZeroDec()

	var optimalRouteLiquidities []types.Dec

	routes, ok := p.priceRoutes[token]
	if !ok {
		return optimalRoute, optimalPrice, optimalRouteLiquidity, nil
	}

	for _, route := range routes {
		price, liquidities, err := p.calculateRoutePrice(height, route, token, decimals)
		if err != nil {
			return nil, types.Dec{}, types.Dec{}, err
		}
		if price.IsZero() {
			// pair exists without any liquidities
			continue
		}
		if len(optimalRoute) == 0 {
			optimalRoute = route
			optimalRouteLiquidities = liquidities
			optimalRouteLiquidity = liquidities[0]
			optimalPrice = price
			continue
		}

		var tmpLiquidities []types.Dec
		if len(liquidities) == len(optimalRouteLiquidities) {
			tmpLiquidities = liquidities
		} else {
			tmpLiquidities = liquidities[:len(optimalRouteLiquidities)]
		}

		isAllEqual := true
		for i, l := range tmpLiquidities {
			if l.GT(optimalRouteLiquidities[i]) {
				optimalRoute = route
				optimalRouteLiquidities = liquidities
				optimalRouteLiquidity = liquidities[0]
				optimalPrice = price

				isAllEqual = false
				break
			} else if l.LT(optimalRouteLiquidities[i]) {
				isAllEqual = false
				break
			}
		}

		if isAllEqual && price.LT(optimalPrice) {
			optimalRoute = route
			optimalRouteLiquidities = liquidities
			optimalPrice = price
		}
	}

	return optimalRoute, optimalPrice, optimalRouteLiquidity, nil
}

func (p *priceImpl) calculateRoutePrice(height uint64, route []string, token string, decimals int64) (types.Dec, []types.Dec, error) {
	liquiditiesInPriceToken := make([]types.Dec, 0)
	price := types.OneDec()

	decimals1 := p.tokenDecimals[p.priceToken]

	// reverse order to derive price from price token
	for i := len(route) - 1; i > -1; i-- {
		asset1 := route[i]

		asset0 := token
		decimals0 := decimals
		if i > 0 {
			asset0 = route[i-1]
			var err error
			decimals0, err = p.decimals(asset0)
			if err != nil {
				return types.Dec{}, nil, errors.Wrap(err, strings.Join([]string{
					"priceImpl.calculateRoutePrice: (Height: ", strconv.FormatUint(height, 10), ")"}, ""))
			}
		}

		liquidity0, liquidity1, err := p.repo.Liquidity(height, asset0, asset1)
		if err != nil {
			return types.Dec{}, nil, errors.Wrap(err, strings.Join([]string{
				"priceImpl.calculateRoutePrice: (Height: ", strconv.FormatUint(height, 10), ")"}, ""))
		}
		liquidity0D, err := util.StringAmountToDecimal(liquidity0, decimals0)
		if err != nil {
			return types.Dec{}, nil, errors.Wrap(err, strings.Join([]string{
				"priceImpl.calculateRoutePrice: (Height: ", strconv.FormatUint(height, 10), ")"}, ""))
		}
		liquidity1D, err := util.StringAmountToDecimal(liquidity1, decimals1)
		if err != nil {
			return types.Dec{}, nil, errors.Wrap(err, strings.Join([]string{
				"priceImpl.calculateRoutePrice: (Height: ", strconv.FormatUint(height, 10), ")"}, ""))
		}

		if liquidity0D.LT(liquidityLowerThreshold) || liquidity1D.LT(liquidityLowerThreshold) {
			return types.ZeroDec(), nil, nil
		}

		liquidityInPriceToken := liquidity1D.MulInt64(2).Mul(price)
		price = liquidity1D.Quo(liquidity0D).Mul(price)
		liquiditiesInPriceToken = append([]types.Dec{liquidityInPriceToken}, liquiditiesInPriceToken...)

		decimals1 = decimals0
	}

	return price, liquiditiesInPriceToken, nil
}
