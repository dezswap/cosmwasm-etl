package router

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/stretchr/testify/require"
)

// srcRepoStub drives Update without a database. UpdateRoutes is only reached when
// the config sets WriteDb, which is how the rebuild is made to fail on demand.
type srcRepoStub struct {
	mutex     sync.Mutex
	pairs     []Pair
	updateErr error
	calls     int
}

func (r *srcRepoStub) Pairs(context.Context) ([]Pair, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.calls++

	return append([]Pair(nil), r.pairs...), nil
}

func (r *srcRepoStub) PairStatus(context.Context) (int, bool, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return len(r.pairs), true, nil
}

func (r *srcRepoStub) UpdateRoutes(context.Context, map[int]string, map[int]map[int][][]int) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.updateErr
}

func (*srcRepoStub) Close() error { return nil }

var _ SrcRepo = &srcRepoStub{}

func testPairs() []Pair {
	return []Pair{
		{Contract: "pair0", AssetInfos: []string{"uusd", "uluna"}},
		{Contract: "pair1", AssetInfos: []string{"uluna", "ukrw"}},
	}
}

func writingRouter(repo SrcRepo) Router {
	return New(repo, configs.RouterConfig{Name: "test", MaxHopCount: 2, WriteDb: true}, logging.Discard)
}

// A failed rebuild used to return while still holding the write lock, wedging every
// later Update forever. The second call below is the regression: it must return
// rather than block.
func TestUpdateReleasesLockWhenRebuildFails(t *testing.T) {
	repo := &srcRepoStub{pairs: testPairs(), updateErr: errors.New("route write failed")}
	router := writingRouter(repo)

	require.Error(t, router.Update(context.Background()))

	done := make(chan error, 1)
	go func() { done <- router.Update(context.Background()) }()

	select {
	case err := <-done:
		require.Error(t, err, "second Update should surface the same failure")
	case <-time.After(5 * time.Second):
		t.Fatal("Update deadlocked: the failed rebuild never released the lock")
	}
}

// The same guarantee for the success path, where the lock is released by the very
// same defer.
func TestUpdateReleasesLockAfterSuccess(t *testing.T) {
	repo := &srcRepoStub{pairs: testPairs()}
	router := writingRouter(repo)

	require.NoError(t, router.Update(context.Background()))

	done := make(chan error, 1)
	go func() { done <- router.Update(context.Background()) }()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Update deadlocked after a successful rebuild")
	}
}

// Readers take the read lock through currentRouteInfo instead of touching the
// routeInfo field directly. Under -race, a reader racing the swap in Update would
// otherwise be reported.
func TestReadersAreSafeWhileUpdateSwapsTheGraph(t *testing.T) {
	repo := &srcRepoStub{pairs: testPairs()}
	router := writingRouter(repo)
	require.NoError(t, router.Update(context.Background()))

	stop := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				// alternate the pair set so shouldUpdate keeps rebuilding
				repo.mutex.Lock()
				if len(repo.pairs) == len(testPairs()) {
					repo.pairs = testPairs()[:1]
				} else {
					repo.pairs = testPairs()
				}
				repo.mutex.Unlock()
				_ = router.Update(context.Background())
			}
		}
	}()

	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					router.Routes("uusd", "ukrw")
					router.TokensFrom("uusd", 2)
				}
			}
		}()
	}

	time.Sleep(200 * time.Millisecond)
	close(stop)
	wg.Wait()
}

// Update reloads only when the pair set actually changed, so a repeated call must
// not rewrite the route table.
func TestUpdateSkipsRebuildWhenPairsAreUnchanged(t *testing.T) {
	repo := &srcRepoStub{pairs: testPairs()}
	router := writingRouter(repo)

	require.NoError(t, router.Update(context.Background()))
	before := router.Routes("uusd", "ukrw")
	require.NotEmpty(t, before, "expected a route across the two pairs")

	require.NoError(t, router.Update(context.Background()))
	require.Equal(t, before, router.Routes("uusd", "ukrw"))
}
