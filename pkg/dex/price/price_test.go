package price

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type transactionRepo struct {
	SrcRepo
	err        error
	txCalled   bool
	rolledBack bool
}

func (r *transactionRepo) WithinTx(_ context.Context, fn func(SrcRepo) error) error {
	r.txCalled = true
	err := fn(r)
	r.rolledBack = err != nil
	return err
}

func (r *transactionRepo) LatestRouteUpdateTimestamp(context.Context) (float64, error) {
	return 0, r.err
}

func TestRunUsesTransactionAndPreservesFailure(t *testing.T) {
	expectedErr := errors.New("route lookup failed")
	repo := &transactionRepo{err: expectedErr}
	tracker := &priceImpl{repo: repo}

	err := tracker.Run(context.Background(), 10)

	require.ErrorIs(t, err, expectedErr)
	require.True(t, repo.txCalled)
	require.True(t, repo.rolledBack)
}

type routeCachingRepo struct {
	SrcRepo
	routeUpdatedTs float64
	routeCalls     int
}

func (r *routeCachingRepo) WithinTx(_ context.Context, fn func(SrcRepo) error) error { return fn(r) }

func (r *routeCachingRepo) LatestRouteUpdateTimestamp(context.Context) (float64, error) {
	return r.routeUpdatedTs, nil
}

func (r *routeCachingRepo) Route(context.Context, string) (map[string][][]string, error) {
	r.routeCalls++
	return map[string][][]string{}, nil
}

func (r *routeCachingRepo) Txs(context.Context, uint64) ([]schemas.ParsedTx, error) {
	return nil, nil
}

func TestRunKeepsRouteCacheAcrossHeights(t *testing.T) {
	repo := &routeCachingRepo{routeUpdatedTs: 100}
	tracker := &priceImpl{repo: repo, priceToken: "uusd"}

	require.NoError(t, tracker.Run(context.Background(), 1))
	require.NoError(t, tracker.Run(context.Background(), 2))
	require.Equal(t, 1, repo.routeCalls, "route table must not be reloaded while the router has published nothing newer")

	repo.routeUpdatedTs = 200
	require.NoError(t, tracker.Run(context.Background(), 3))
	require.Equal(t, 2, repo.routeCalls, "a newer router update must invalidate the cache")
}

type skippingRepo struct {
	SrcRepo
	decimalsErr  map[string]error
	directWrites int
}

func (r *skippingRepo) WithinTx(_ context.Context, fn func(SrcRepo) error) error { return fn(r) }

func (r *skippingRepo) LatestRouteUpdateTimestamp(context.Context) (float64, error) { return 0, nil }

func (r *skippingRepo) Route(context.Context, string) (map[string][][]string, error) {
	return map[string][][]string{}, nil
}

func (r *skippingRepo) Txs(context.Context, uint64) ([]schemas.ParsedTx, error) {
	// a direct swap against the price token, so the price hinges on asset1's decimals
	return []schemas.ParsedTx{{
		Height: 10, Id: 1, Hash: "hash", Asset0: "uusd", Asset1: "B",
		Asset0Amount: "1000000", Asset1Amount: "2000000",
	}}, nil
}

func (r *skippingRepo) Decimals(_ context.Context, token string) (int64, error) {
	if err, ok := r.decimalsErr[token]; ok {
		return 0, err
	}
	return 6, nil
}

func (r *skippingRepo) UpdateDirectPrice(context.Context, uint64, uint64, string, string, string, bool) error {
	r.directWrites++
	return nil
}

func newSkipTracker(repo SrcRepo) *priceImpl {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	return &priceImpl{repo: repo, priceToken: "uusd", logger: logger, tokenDecimals: map[string]int64{"uusd": 6}}
}

func TestRunSkipsUnregisteredTokenInsteadOfFailingTheHeight(t *testing.T) {
	// a failed height ends the price scheduler, which cancels every other task, and
	// the restart meets the same height again
	repo := &skippingRepo{decimalsErr: map[string]error{"B": fmt.Errorf("lookup: %w", ErrTokenNotFound)}}
	tracker := newSkipTracker(repo)

	require.NoError(t, tracker.Run(context.Background(), 10))
	require.Zero(t, repo.directWrites, "a token of unknown decimals must not be priced")
	require.Equal(t, uint64(1), tracker.skips["B"].count)
}

func TestNoteBlockerWarnsWithoutCountingAMissedPrice(t *testing.T) {
	tracker := newSkipTracker(&skippingRepo{})

	// optimalRoutePrice tries every route of a token, so counting here would tally
	// route attempts; the missed price is tallied once the token runs out of routes
	tracker.noteBlocker("B", 10, ErrTokenNotFound)
	tracker.noteBlocker("B", 11, ErrTokenNotFound)
	require.Zero(t, tracker.skips["B"].count)

	tracker.recordSkip("B", 12, ErrTokenNotFound)
	require.Equal(t, uint64(1), tracker.skips["B"].count)
	require.Equal(t, uint64(10), tracker.skips["B"].firstHeight, "the ledger keeps where it started")
}

func TestRunPropagatesUnexpectedDecimalsFailure(t *testing.T) {
	expectedErr := errors.New("connection reset")
	repo := &skippingRepo{decimalsErr: map[string]error{"B": expectedErr}}
	tracker := newSkipTracker(repo)

	require.ErrorIs(t, tracker.Run(context.Background(), 10), expectedErr,
		"only a missing token or route may be skipped, never a broken query")
}

type routePriceRepo struct {
	SrcRepo
	decimalsErr   map[string]error
	updateErr     error
	updatedTokens []string
}

func (r *routePriceRepo) Decimals(_ context.Context, token string) (int64, error) {
	if err, ok := r.decimalsErr[token]; ok {
		return 0, err
	}
	return 6, nil
}

func (r *routePriceRepo) Liquidity(context.Context, uint64, string, string) (string, string, error) {
	return "100000000", "100000000", nil
}

func (r *routePriceRepo) UpdateRoutePrice(_ context.Context, _ uint64, _ uint64, token string, _ string, _ string, route []string) error {
	if len(route) == 0 {
		return ErrRouteNotFound
	}
	if r.updateErr != nil {
		return r.updateErr
	}
	r.updatedTokens = append(r.updatedTokens, token)
	return nil
}

// routeDropRepo prices every hop from a fixed liquidity, overridable per token, so a
// route can be made illiquid or blocked by an unregistered hop.
type routeDropRepo struct {
	SrcRepo
	decimalsErr map[string]error
	liquidity   map[string]string
	written     []string
}

func (r *routeDropRepo) Decimals(_ context.Context, token string) (int64, error) {
	if err, ok := r.decimalsErr[token]; ok {
		return 0, err
	}
	return 6, nil
}

func (r *routeDropRepo) Liquidity(_ context.Context, _ uint64, token string, _ string) (string, string, error) {
	if l, ok := r.liquidity[token]; ok {
		return l, l, nil
	}
	return "100000000", "100000000", nil
}

func (r *routeDropRepo) UpdateRoutePrice(_ context.Context, _ uint64, _ uint64, token string, _ string, _ string, _ []string) error {
	r.written = append(r.written, token)
	return nil
}

func indirectSwap() schemas.ParsedTx {
	return schemas.ParsedTx{
		Height: 10, Id: 1, Hash: "hash", Asset0: "A", Asset1: "B",
		Asset0Amount: "1000000", Asset1Amount: "2000000",
	}
}

func TestUpdateIndirectSwapPriceSeparatesWhyNoRoutePriced(t *testing.T) {
	// "1000000" at 6 decimals is 1.0, under liquidityLowerThreshold
	illiquid := map[string]string{"A": "1000000", "B": "1000000"}

	t.Run("illiquid route is not an unpublished one", func(t *testing.T) {
		repo := &routeDropRepo{liquidity: illiquid}
		tracker := newSkipTracker(repo)
		tracker.priceRoutes = map[string][][]string{"A": {{"uusd"}}, "B": {{"uusd"}}}

		require.NoError(t, tracker.updateIndirectSwapPrice(context.Background(), repo, indirectSwap()))

		require.Empty(t, repo.written)
		// the route exists, and no backfill will ever price this height differently
		require.Equal(t, "route liquidity below threshold", tracker.skips["A"].reason)
		require.Equal(t, "route liquidity below threshold", tracker.skips["B"].reason)
	})

	t.Run("token without a route definition is unpublished", func(t *testing.T) {
		repo := &routeDropRepo{}
		tracker := newSkipTracker(repo)
		tracker.priceRoutes = map[string][][]string{}

		require.NoError(t, tracker.updateIndirectSwapPrice(context.Background(), repo, indirectSwap()))

		require.Equal(t, "unpublished route", tracker.skips["A"].reason)
		require.Equal(t, "unpublished route", tracker.skips["B"].reason)
	})

	t.Run("unregistered hop is blamed on the hop, and still tallied on the token", func(t *testing.T) {
		// A's only route runs through X, which has no row in tokens. Calling that an
		// unpublished route points at the wrong thing; recording nothing hides a price
		// a backfill can still reach.
		repo := &routeDropRepo{decimalsErr: map[string]error{"X": fmt.Errorf("lookup: %w", ErrTokenNotFound)}}
		tracker := newSkipTracker(repo)
		tracker.priceRoutes = map[string][][]string{"A": {{"X", "uusd"}}}

		require.NoError(t, tracker.updateIndirectSwapPrice(context.Background(), repo, indirectSwap()))

		require.Equal(t, "unregistered token", tracker.skips["A"].reason)
		require.Equal(t, uint64(1), tracker.skips["A"].count)
		require.Zero(t, tracker.skips["X"].count, "the hop is named, not charged a missed price")
		require.Equal(t, "unpublished route", tracker.skips["B"].reason)
	})

	t.Run("unregistered hop outranks an illiquid route", func(t *testing.T) {
		repo := &routeDropRepo{
			decimalsErr: map[string]error{"X": fmt.Errorf("lookup: %w", ErrTokenNotFound)},
			liquidity:   map[string]string{"Y": "1000000"},
		}
		tracker := newSkipTracker(repo)
		tracker.priceRoutes = map[string][][]string{"A": {{"X", "uusd"}, {"Y", "uusd"}}}

		require.NoError(t, tracker.updateIndirectSwapPrice(context.Background(), repo, indirectSwap()))

		// the actionable reason must survive a later route dying for another cause
		require.Equal(t, "unregistered token", tracker.skips["A"].reason)
	})
}

func TestRecoverableSeparatesBackfillableReasons(t *testing.T) {
	require.True(t, recoverable(ErrTokenNotFound))
	require.True(t, recoverable(ErrRouteNotFound))
	require.False(t, recoverable(ErrRouteIlliquid), "liquidity at a height never changes")
}

func TestUpdateIndirectSwapPriceSkipsAssetWithoutRoute(t *testing.T) {
	repo := &routePriceRepo{}
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	tracker := &priceImpl{
		repo:          repo,
		priceToken:    "uusd",
		logger:        logger,
		tokenDecimals: map[string]int64{"uusd": 6},
		// only asset0 is routable to the price token
		priceRoutes: map[string][][]string{"A": {{"B", "uusd"}}},
	}

	err := tracker.updateIndirectSwapPrice(context.Background(), repo, schemas.ParsedTx{
		Height: 10, Id: 1, Hash: "hash", Asset0: "A", Asset1: "B",
		Asset0Amount: "1000000", Asset1Amount: "2000000",
	})

	require.NoError(t, err)
	require.Equal(t, []string{"A"}, repo.updatedTokens, "an asset without a route must not be written as a price row")
}

func TestUpdateIndirectSwapPricePricesIndependentCounterpartWithMissingDecimals(t *testing.T) {
	tests := []struct {
		name         string
		asset0       string
		asset1       string
		missingToken string
		pricedToken  string
	}{
		{name: "asset0 decimals missing", asset0: "A", asset1: "B", missingToken: "A", pricedToken: "B"},
		{name: "asset1 decimals missing", asset0: "A", asset1: "B", missingToken: "B", pricedToken: "A"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &routePriceRepo{decimalsErr: map[string]error{
				tt.missingToken: fmt.Errorf("lookup: %w", ErrTokenNotFound),
			}}
			tracker := newSkipTracker(repo)
			tracker.priceRoutes = map[string][][]string{
				tt.pricedToken: {{tracker.priceToken}},
			}

			err := tracker.updateIndirectSwapPrice(context.Background(), repo, schemas.ParsedTx{
				Height: 10, Id: 1, Hash: "hash", Asset0: tt.asset0, Asset1: tt.asset1,
				Asset0Amount: "1000000", Asset1Amount: "2000000",
			})

			require.NoError(t, err)
			require.Equal(t, []string{tt.pricedToken}, repo.updatedTokens)
			require.Len(t, tracker.skips, 1)
			require.Contains(t, tracker.skips, tt.missingToken)
			require.Equal(t, uint64(1), tracker.skips[tt.missingToken].count)
		})
	}
}

func TestUpdateIndirectSwapPriceDoesNotRecordMissingDecimalsWhenCounterpartWriteFails(t *testing.T) {
	expectedErr := errors.New("write failed")
	repo := &routePriceRepo{
		decimalsErr: map[string]error{"A": fmt.Errorf("lookup: %w", ErrTokenNotFound)},
		updateErr:   expectedErr,
	}
	tracker := newSkipTracker(repo)
	tracker.priceRoutes = map[string][][]string{"B": {{tracker.priceToken}}}

	err := tracker.updateIndirectSwapPrice(context.Background(), repo, schemas.ParsedTx{
		Height: 10, Id: 1, Hash: "hash", Asset0: "A", Asset1: "B",
		Asset0Amount: "1000000", Asset1Amount: "2000000",
	})

	require.ErrorIs(t, err, expectedErr)
	require.Empty(t, tracker.skips, "a failed height must not update the non-transactional skip ledger")
}

func TestCalculatePrice(t *testing.T) {
	assert := assert.New(t)

	p := &priceImpl{}
	dec, err := p.calculatePrice("2000000000000000000", 18, "1000000", 6, false)

	assert.NoError(err)
	assert.Equal(dec.String(), "0.500000000000000000")
}

func TestCalculatePrice_Negative(t *testing.T) {
	assert := assert.New(t)

	p := &priceImpl{}
	dec, err := p.calculatePrice("-2000000000000000000", 18, "1000000", 6, false)

	assert.NoError(err)
	assert.Equal(dec.String(), "0.500000000000000000")
}
