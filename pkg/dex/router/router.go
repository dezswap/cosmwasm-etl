package router

import (
	"github.com/dezswap/cosmwasm-etl/configs"
	"sort"
	"sync"

	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/pkg/errors"
)

type Router interface {
	RouterAddress() string
	Routes(from, to string) [][]string
	TokensFrom(from string, hopCount int) []string
	Update() error
}

var _ Router = &routerImpl{}

type routerImpl struct {
	name          string
	repo          SrcRepo
	logger        logging.Logger
	routerAddress string
	maxPathLen    uint
	writeDb       bool

	// state
	cachedPairs []Pair
	routeInfo
	mutex *sync.Mutex
}

func New(repo SrcRepo, c configs.RouterConfig, logger logging.Logger) (Router, error) {
	return &routerImpl{
		name:          c.Name,
		logger:        logger,
		repo:          repo,
		routerAddress: c.RouterAddr,
		mutex:         &sync.Mutex{},
		maxPathLen:    c.MaxPathLen,
		writeDb:       c.WriteDb,
	}, nil
}

func (r *routerImpl) RouterAddress() string {
	return r.routerAddress
}

func (r *routerImpl) TokensFrom(from string, hopCount int) []string {
	routeInfo := r.routeInfo
	if routeInfo == nil {
		return nil
	}

	fromIdx, fromOk := routeInfo.indexFromAddress(from)
	pathLen := hopCount + 1

	destMap, ok := routeInfo.routesMapOf(fromIdx)
	if !fromOk || !ok {
		return nil
	}
	tokenMap := make(map[string]bool)

	for k, paths := range destMap {
		for _, path := range paths {
			if len(path) > pathLen {
				continue
			}
			tokenMap[routeInfo.addressFromIndex(k)] = true
		}
	}

	tokens := make([]string, 0, len(tokenMap))
	for k := range tokenMap {
		tokens = append(tokens, k)
	}
	sort.Strings(tokens)
	return tokens
}

func (r *routerImpl) Routes(from, to string) [][]string {
	cachedInfo := r.routeInfo
	if cachedInfo == nil {
		return nil
	}
	fromIdx, fromOk := cachedInfo.indexFromAddress(from)
	toIdx, toOk := cachedInfo.indexFromAddress(to)
	if !fromOk || !toOk || cachedInfo.pairsMapOf(fromIdx) == nil || cachedInfo.routesIndexesOf(fromIdx, toIdx) == nil {
		return nil
	}

	routesIndexes := cachedInfo.routesIndexesOf(fromIdx, toIdx)
	routesArr := [][]string{}
	for _, routeIndexes := range routesIndexes {
		hopAddresses := []string{}
		for _, routeIdx := range routeIndexes {
			hopAddresses = append(hopAddresses, cachedInfo.addressFromIndex(routeIdx))
		}
		routesArr = append(routesArr, hopAddresses)
	}
	return routesArr
}

func (r *routerImpl) Update() error {
	pairs, err := r.repo.Pairs()
	if err != nil {
		return errors.Wrap(err, "routerImpl.Update")
	}

	r.mutex.Lock()

	if r.shouldUpdate(pairs) {
		var repo SrcRepo
		if r.writeDb {
			repo = r.repo
		}
		ri, err := newRouteInfo(pairs, r.maxPathLen, repo)
		if err != nil {
			return err
		}

		r.routeInfo = ri
		r.cachedPairs = pairs
	}

	r.mutex.Unlock()
	return nil
}

func (r *routerImpl) shouldUpdate(pairs []Pair) bool {
	if len(r.cachedPairs) != len(pairs) {
		return true
	}
	lmt := len(pairs)
	for idx := 0; idx < lmt; idx++ {
		if pairs[idx].Contract != r.cachedPairs[idx].Contract {
			return true
		}
	}
	return false
}
