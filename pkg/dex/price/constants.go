package price

import (
	"cosmossdk.io/math"
	"github.com/pkg/errors"
)

const NaValue = int64(-1)

var liquidityLowerThreshold = math.LegacyNewDec(10)

var (
	// ErrTokenNotFound reports a token missing from the tokens table. Treating it as
	// decimals = 0 would put every price derived from it out by orders of magnitude.
	ErrTokenNotFound = errors.New("token not found")
	// ErrRouteNotFound reports that no route row matches the price about to be
	// written, which would otherwise be stored with zeroed token and route ids.
	ErrRouteNotFound = errors.New("price route not found")
	// ErrRouteIlliquid reports that every route of a token holds too little liquidity
	// to price it. Unlike the two above it never clears, so a backfill cannot recover
	// the prices missed for it.
	ErrRouteIlliquid = errors.New("price route liquidity below threshold")
)
