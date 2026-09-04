package price

import (
	"context"
	"strconv"
	"strings"
	"time"

	"cosmossdk.io/math"
	"github.com/pkg/errors"

	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/dezswap/cosmwasm-etl/pkg/util"
)

type Price interface {
	SrcHeight(context.Context) (int64, error)
	NextHeight(context.Context, uint64) (int64, error)
	Run(context.Context, uint64) error
}

var _ Price = &priceImpl{}

type priceImpl struct {
	repo       SrcRepo
	priceToken string
	logger     logging.Logger

	tokenDecimals               map[string]int64
	priceRoutes                 map[string][][]string
	latestRouteUpdatedTimestamp time.Time
	skips                       map[string]*skipRecord
}

// skipRecord is what became of one token the task could not price. count holds the
// prices actually missed, which is not every mention: see noteBlocker.
type skipRecord struct {
	reason      string
	firstHeight uint64
	count       uint64
}

func New(ctx context.Context, repo SrcRepo, priceToken string, logger logging.Logger) (Price, error) {
	// returning the call through would hand back a non nil Price holding a nil
	// *priceImpl on failure
	p, err := newPriceImpl(ctx, repo, priceToken, logger)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// newPriceImpl hands back the concrete tracker, which Backfill needs for its skip
// ledger.
func newPriceImpl(ctx context.Context, repo SrcRepo, priceToken string, logger logging.Logger) (*priceImpl, error) {
	tokenDecimals := make(map[string]int64)
	priceTokenDecimal, err := repo.Decimals(ctx, priceToken)
	if err != nil {
		// every price of the chain is denominated in it; there is nothing to skip
		return nil, errors.Wrapf(err, "price.New: price token(%s) decimals", priceToken)
	}
	tokenDecimals[priceToken] = priceTokenDecimal

	p := &priceImpl{
		logger:        logger,
		priceToken:    priceToken,
		repo:          repo,
		tokenDecimals: tokenDecimals,
	}

	return p, nil
}

func (p *priceImpl) SrcHeight(ctx context.Context) (int64, error) {
	height, err := p.repo.SrcHeight(ctx)
	if err != nil {
		return NaValue, err
	}

	return height, nil
}

func (p *priceImpl) NextHeight(ctx context.Context, minHeight uint64) (int64, error) {
	if minHeight == 0 {
		if firstHeight, err := p.repo.FirstHeight(ctx, p.priceToken); err != nil {
			return NaValue, err
		} else if firstHeight > 0 {
			minHeight = uint64(firstHeight) - 1
		}
	}
	height, err := p.repo.NextHeight(ctx, minHeight)
	if err != nil {
		return NaValue, err
	}

	return height, nil
}

// Run calculates one height inside a single transaction, so a failed or canceled
// height is retried as a whole. The tx-scoped repository is passed down explicitly
// to keep cached routes and decimals on p across heights.
func (p *priceImpl) Run(ctx context.Context, height uint64) error {
	return p.repo.WithinTx(ctx, func(txRepo SrcRepo) error {
		return p.run(ctx, txRepo, height)
	})
}

func (p *priceImpl) run(ctx context.Context, repo SrcRepo, height uint64) error {
	if err := p.updatePriceRoute(ctx, repo); err != nil {
		return err
	}

	txs, err := repo.Txs(ctx, height)
	if err != nil {
		return err
	}

	for _, t := range txs {
		if t.Asset0Amount == "0" || t.Asset1Amount == "0" {
			continue
		}

		if t.Asset0 == p.priceToken || t.Asset1 == p.priceToken {
			if err := p.updateDirectSwapPrice(ctx, repo, t); err != nil {
				return err
			}
		} else {
			if err := p.updateIndirectSwapPrice(ctx, repo, t); err != nil {
				return err
			}
		}
	}

	return nil
}

// updatePriceRoute reloads the route table only when the router has published a
// newer one.
func (p *priceImpl) updatePriceRoute(ctx context.Context, repo SrcRepo) error {
	ts, err := repo.LatestRouteUpdateTimestamp(ctx)
	if err != nil {
		return err
	}
	lruts := util.ToTime(ts)
	if lruts.After(p.latestRouteUpdatedTimestamp) {
		p.priceRoutes, err = repo.Route(ctx, p.priceToken)
		if err != nil {
			return err
		}
		p.latestRouteUpdatedTimestamp = lruts
	}

	return nil
}

func (p *priceImpl) updateDirectSwapPrice(ctx context.Context, repo SrcRepo, tx schemas.ParsedTx) error {
	isReverse := tx.Asset0 == p.priceToken

	var targetToken string
	var decimals0, decimals1 int64
	var err error

	if isReverse {
		targetToken = tx.Asset1
		decimals0 = p.tokenDecimals[p.priceToken]
		decimals1, err = p.decimals(ctx, repo, tx.Asset1)
	} else {
		targetToken = tx.Asset0
		decimals0, err = p.decimals(ctx, repo, tx.Asset0)
		decimals1 = p.tokenDecimals[p.priceToken]
	}
	if err != nil {
		if !skippable(err) {
			return err
		}
		p.recordSkip(targetToken, tx.Height, err)
		return nil
	}

	price, err := p.calculatePrice(tx.Asset0Amount, decimals0, tx.Asset1Amount, decimals1, isReverse)
	if err != nil {
		return errors.Wrap(err, strings.Join([]string{
			"priceImpl.updateDirectSwapPrice: (Tx Hash: ", tx.Hash, ")"}, ""))
	}

	if err := repo.UpdateDirectPrice(ctx, tx.Height, tx.Id, targetToken, price.String(), p.priceToken, isReverse); err != nil {
		if !skippable(err) {
			return err
		}
		// a pair that started trading before the router published its route lands here
		p.recordSkip(targetToken, tx.Height, err)
	}

	return nil
}

func (p *priceImpl) calculatePrice(asset0Amount string, asset0Decimals int64, asset1Amount string, asset1Decimals int64, isReverse bool) (math.LegacyDec, error) {
	asset0AmountD, err := util.StringAmountToDecimal(asset0Amount, asset0Decimals)
	if err != nil {
		return math.LegacyDec{}, err
	}
	asset1AmountD, err := util.StringAmountToDecimal(asset1Amount, asset1Decimals)
	if err != nil {
		return math.LegacyDec{}, err
	}

	if isReverse {
		return asset0AmountD.Quo(asset1AmountD).Abs(), nil
	}
	return asset1AmountD.Quo(asset0AmountD).Abs(), nil
}

func (p *priceImpl) decimals(ctx context.Context, repo SrcRepo, token string) (int64, error) {
	var decimals int64
	var err error

	if d, ok := p.tokenDecimals[token]; ok {
		decimals = d
	} else {
		decimals, err = repo.Decimals(ctx, token)
		if err != nil {
			return NaValue, err
		}
		p.tokenDecimals[token] = decimals
	}

	return decimals, nil
}

// skippable reports whether err is a prerequisite that has not landed yet rather than
// a failure. Both resolve on their own, and the prices passed over meanwhile are
// recoverable with cmd/aggregator/pricebackfill. Failing instead would end the price
// scheduler, which cancels the errgroup holding every other task, and the restart would
// meet the same height again.
func skippable(err error) bool {
	return errors.Is(err, ErrTokenNotFound) || errors.Is(err, ErrRouteNotFound)
}

// routeUnusable reports whether err leaves a route unable to price a token rather than
// meaning the calculation failed.
func routeUnusable(err error) bool {
	return skippable(err) || errors.Is(err, ErrRouteIlliquid)
}

func skipReason(err error) string {
	switch {
	case errors.Is(err, ErrTokenNotFound):
		return "unregistered token"
	case errors.Is(err, ErrRouteIlliquid):
		return "route liquidity below threshold"
	default:
		return "unpublished route"
	}
}

// recoverable reports whether a backfill can still produce the prices missed for this
// reason.
func recoverable(err error) bool {
	return !errors.Is(err, ErrRouteIlliquid)
}

// skipEntry returns token's ledger entry, warning the first time the token turns up.
// One unregistered token can appear in thousands of swaps, so the log names it once.
func (p *priceImpl) skipEntry(token string, height uint64, err error) *skipRecord {
	if p.skips == nil {
		p.skips = make(map[string]*skipRecord)
	}
	if record, ok := p.skips[token]; ok {
		return record
	}

	reason := skipReason(err)
	record := &skipRecord{reason: reason, firstHeight: height}
	p.skips[token] = record

	advice := ""
	if recoverable(err) {
		advice = "; backfill it once resolved"
	}
	p.logger.Warnf("token(%s) cannot be priced from height %d onwards, %s%s: %s",
		token, height, reason, advice, err)

	return record
}

// recordSkip tallies one price that could not be written.
func (p *priceImpl) recordSkip(token string, height uint64, err error) {
	p.skipEntry(token, height, err).count++
}

// noteBlocker reports a token that makes a route unusable without tallying a missed
// price. A dead route is not a missed price on its own: optimalRoutePrice tries the
// others, and if none survive, the token being priced is tallied by writeRoutePrice.
// Tallying here instead would count route attempts, and against the wrong token.
func (p *priceImpl) noteBlocker(token string, height uint64, err error) {
	p.skipEntry(token, height, err)
}

func (p *priceImpl) updateIndirectSwapPrice(ctx context.Context, repo SrcRepo, tx schemas.ParsedTx) error {
	decimals0, decimals0Err := p.decimals(ctx, repo, tx.Asset0)
	if decimals0Err != nil && !skippable(decimals0Err) {
		return errors.Wrap(decimals0Err,
			strings.Join([]string{"priceImpl.updateIndirectSwapPrice: (Tx hash:", tx.Hash, ")"}, ""))
	}
	decimals1, decimals1Err := p.decimals(ctx, repo, tx.Asset1)
	if decimals1Err != nil && !skippable(decimals1Err) {
		return errors.Wrap(decimals1Err,
			strings.Join([]string{"priceImpl.updateIndirectSwapPrice: (Tx hash:", tx.Hash, ")"}, ""))
	}

	var route0, route1 []string
	price0, price1 := math.LegacyZeroDec(), math.LegacyZeroDec()
	liquidity0, liquidity1 := math.LegacyZeroDec(), math.LegacyZeroDec()
	var drop0, drop1 error

	if decimals0Err == nil {
		var err error
		route0, price0, liquidity0, err = p.optimalRoutePrice(ctx, repo, tx.Height, tx.Asset0, decimals0)
		if err != nil {
			if !routeUnusable(err) {
				return errors.Wrap(err,
					strings.Join([]string{"priceImpl.updateIndirectSwapPrice: (Tx hash:", tx.Hash, ")"}, ""))
			}
			drop0 = err
		}
	}
	if decimals1Err == nil {
		var err error
		route1, price1, liquidity1, err = p.optimalRoutePrice(ctx, repo, tx.Height, tx.Asset1, decimals1)
		if err != nil {
			if !routeUnusable(err) {
				return errors.Wrap(err,
					strings.Join([]string{"priceImpl.updateIndirectSwapPrice: (Tx hash:", tx.Hash, ")"}, ""))
			}
			drop1 = err
		}
	}

	// Swap-ratio adjustments need both assets' decimals. When only one asset is
	// registered, its independent route can still be priced and written below.
	if decimals0Err != nil || decimals1Err != nil {
		if decimals0Err == nil {
			if err := p.writeRoutePrice(ctx, repo, tx, tx.Asset0, price0, route0, drop0); err != nil {
				return err
			}
		}
		if decimals1Err == nil {
			if err := p.writeRoutePrice(ctx, repo, tx, tx.Asset1, price1, route1, drop1); err != nil {
				return err
			}
		}

		// The skip ledger is not part of the repository transaction. Update it only
		// after every usable counterpart has been written successfully, so a failed
		// height does not leave a skip behind when the database rolls back.
		if decimals0Err != nil {
			p.recordSkip(tx.Asset0, tx.Height, decimals0Err)
		}
		if decimals1Err != nil {
			p.recordSkip(tx.Asset1, tx.Height, decimals1Err)
		}
		return nil
	}

	if len(route0) == 0 && len(route1) == 0 {
		p.recordSkip(tx.Asset0, tx.Height, drop0)
		p.recordSkip(tx.Asset1, tx.Height, drop1)
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

	// an asset can still be left without a route here, e.g. when the counterpart route
	// already contains it
	if err := p.writeRoutePrice(ctx, repo, tx, tx.Asset0, price0, route0, drop0); err != nil {
		return err
	}

	return p.writeRoutePrice(ctx, repo, tx, tx.Asset1, price1, route1, drop1)
}

// writeRoutePrice stores one asset's routed price. An asset no route could price is
// recorded under dropped, which optimalRoutePrice sets whenever it returns no route.
func (p *priceImpl) writeRoutePrice(ctx context.Context, repo SrcRepo, tx schemas.ParsedTx, token string, price math.LegacyDec, route []string, dropped error) error {
	if len(route) == 0 {
		p.recordSkip(token, tx.Height, dropped)
		return nil
	}

	if err := repo.UpdateRoutePrice(ctx, tx.Height, tx.Id, token, price.Abs().String(), p.priceToken, route); err != nil {
		if !skippable(err) {
			return err
		}
		p.recordSkip(token, tx.Height, err)
	}

	return nil
}

// optimalRoutePrice picks the best priced route for a token. When none can price it,
// route is empty and the error says why: ErrRouteNotFound, ErrRouteIlliquid or
// ErrTokenNotFound. routeUnusable tells those apart from a failure.
func (p *priceImpl) optimalRoutePrice(ctx context.Context, repo SrcRepo, height uint64, token string, decimals int64) ([]string, math.LegacyDec, math.LegacyDec, error) {
	var optimalRoute []string
	optimalPrice := math.LegacyZeroDec()
	optimalRouteLiquidity := math.LegacyZeroDec()

	var optimalRouteLiquidities []math.LegacyDec

	routes, ok := p.priceRoutes[token]
	if !ok {
		return nil, optimalPrice, optimalRouteLiquidity, ErrRouteNotFound
	}

	drop := ErrRouteNotFound
	for _, route := range routes {
		price, liquidities, err := p.calculateRoutePrice(ctx, repo, height, route, token, decimals)
		if err != nil {
			if !routeUnusable(err) {
				return nil, math.LegacyDec{}, math.LegacyDec{}, err
			}
			// an unregistered hop outranks illiquidity: only it is actionable
			if !errors.Is(drop, ErrTokenNotFound) {
				drop = err
			}
			continue
		}
		if len(optimalRoute) == 0 {
			optimalRoute = route
			optimalRouteLiquidities = liquidities
			optimalRouteLiquidity = liquidities[0]
			optimalPrice = price
			continue
		}

		var tmpLiquidities []math.LegacyDec
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

	if len(optimalRoute) == 0 {
		return nil, optimalPrice, optimalRouteLiquidity, drop
	}

	return optimalRoute, optimalPrice, optimalRouteLiquidity, nil
}

// calculateRoutePrice prices one route, reporting ErrTokenNotFound or ErrRouteIlliquid
// when the route cannot price the token at this height.
func (p *priceImpl) calculateRoutePrice(ctx context.Context, repo SrcRepo, height uint64, route []string, token string, decimals int64) (math.LegacyDec, []math.LegacyDec, error) {
	liquiditiesInPriceToken := make([]math.LegacyDec, 0)
	price := math.LegacyOneDec()

	decimals1 := p.tokenDecimals[p.priceToken]

	// reverse order to derive price from price token
	for i := len(route) - 1; i > -1; i-- {
		asset1 := route[i]

		asset0 := token
		decimals0 := decimals
		if i > 0 {
			asset0 = route[i-1]
			var err error
			decimals0, err = p.decimals(ctx, repo, asset0)
			if err != nil {
				if !skippable(err) {
					return math.LegacyDec{}, nil, errors.Wrap(err, strings.Join([]string{
						"priceImpl.calculateRoutePrice: (Height: ", strconv.FormatUint(height, 10), ")"}, ""))
				}
				p.noteBlocker(asset0, height, err)
				return math.LegacyDec{}, nil, err
			}
		}

		liquidity0, liquidity1, err := repo.Liquidity(ctx, height, asset0, asset1)
		if err != nil {
			return math.LegacyDec{}, nil, errors.Wrap(err, strings.Join([]string{
				"priceImpl.calculateRoutePrice: (Height: ", strconv.FormatUint(height, 10), ")"}, ""))
		}
		liquidity0D, err := util.StringAmountToDecimal(liquidity0, decimals0)
		if err != nil {
			return math.LegacyDec{}, nil, errors.Wrap(err, strings.Join([]string{
				"priceImpl.calculateRoutePrice: (Height: ", strconv.FormatUint(height, 10), ")"}, ""))
		}
		liquidity1D, err := util.StringAmountToDecimal(liquidity1, decimals1)
		if err != nil {
			return math.LegacyDec{}, nil, errors.Wrap(err, strings.Join([]string{
				"priceImpl.calculateRoutePrice: (Height: ", strconv.FormatUint(height, 10), ")"}, ""))
		}

		if liquidity0D.LT(liquidityLowerThreshold) || liquidity1D.LT(liquidityLowerThreshold) {
			return math.LegacyDec{}, nil, ErrRouteIlliquid
		}

		liquidityInPriceToken := liquidity1D.MulInt64(2).Mul(price)
		price = liquidity1D.Quo(liquidity0D).Mul(price)
		liquiditiesInPriceToken = append([]math.LegacyDec{liquidityInPriceToken}, liquiditiesInPriceToken...)

		decimals1 = decimals0
	}

	return price, liquiditiesInPriceToken, nil
}
