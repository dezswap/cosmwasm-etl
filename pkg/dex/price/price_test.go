package price

import (
	"context"
	"errors"
	"testing"

	"github.com/dezswap/cosmwasm-etl/pkg/db/schemas"
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
