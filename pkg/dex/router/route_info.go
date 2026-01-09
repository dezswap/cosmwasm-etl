package router

import (
	"sort"
)

type routeInfo interface {
	pairsMapOf(fromIdx int) map[int]string
	routesMapOf(fromIdx int) (destMap map[int][][]int, ok bool)
	routesIndexesOf(fromIdx, toIdx int) [][]int

	addressFromIndex(idx int) string
	indexFromAddress(from string) (v int, ok bool)
}

type routeInfoImpl struct {
	maxHopCount uint
	// from, to, pair address
	initialPairMap map[int]map[int]string
	// from, to, path indexes
	routesMap map[int]map[int][][]int

	// to reduce memory usage
	indexToAsset map[int]string
	assetToIndex map[string]int
}

var _ routeInfo = &routeInfoImpl{}

// newRouteInfo implements cache
func newRouteInfo(pairs []Pair, maxHopCount uint, repo SrcRepo) (routeInfo, error) {
	if maxHopCount > MAX_ROUTE_HOP_COUNT {
		maxHopCount = MAX_ROUTE_HOP_COUNT
	}
	ri := routeInfoImpl{maxHopCount: maxHopCount}

	ri.setIndex(pairs)
	ri.setPairMap(pairs)
	if err := ri.setRoutesMap(repo); err != nil {
		return nil, err
	}

	return &ri, nil
}

// routesIndexesOf implements routeInfo
func (ri *routeInfoImpl) routesIndexesOf(fromIdx int, toIdx int) [][]int {
	return ri.routesMap[fromIdx][toIdx]
}

// routesMapOf implements routeInfo
func (ri *routeInfoImpl) routesMapOf(fromIdx int) (destMap map[int][][]int, ok bool) {
	destMap, ok = ri.routesMap[fromIdx]
	return destMap, ok
}

// pairsMapOf implements routeInfo
func (ri *routeInfoImpl) pairsMapOf(fromIdx int) map[int]string {
	return ri.initialPairMap[fromIdx]
}

func (ri *routeInfoImpl) addressFromIndex(idx int) string {
	return ri.indexToAsset[idx]
}

func (ri *routeInfoImpl) indexFromAddress(from string) (v int, ok bool) {
	v, ok = ri.assetToIndex[from]
	return v, ok
}

func (ri *routeInfoImpl) setIndex(pairs []Pair) {
	ri.indexToAsset = make(map[int]string)
	ri.assetToIndex = make(map[string]int)

	idx := 0
	for _, pair := range pairs {
		for _, token := range pair.AssetInfos {
			if _, ok := ri.assetToIndex[token]; ok {
				continue
			}

			ri.assetToIndex[token] = idx
			ri.indexToAsset[idx] = token
			idx++
		}
	}
}

func (ri *routeInfoImpl) setPairMap(pairs []Pair) {
	ri.initialPairMap = make(map[int]map[int]string)
	for _, p := range pairs {
		first := p.AssetInfos[0]
		second := p.AssetInfos[1]

		from := ri.assetToIndex[first]
		if ri.initialPairMap[from] == nil {
			ri.initialPairMap[from] = make(map[int]string)
		}
		to := ri.assetToIndex[second]
		if ri.initialPairMap[to] == nil {
			ri.initialPairMap[to] = make(map[int]string)
		}
		ri.initialPairMap[from][to] = p.Contract
		ri.initialPairMap[to][from] = p.Contract
	}
}

func (ri *routeInfoImpl) setRoutesMap(repo SrcRepo) error {
	ri.routesMap = make(map[int]map[int][][]int)
	keys := make([]int, 0, len(ri.initialPairMap))
	visited := make(map[int]bool)
	for k := range ri.initialPairMap {
		keys = append(keys, k)
		visited[k] = false
	}

	for _, key := range keys {
		ri.routesMap[key] = make(map[int][][]int)
		visited[key] = true
		ri.findAllRoutes(key, key, []int{}, visited, 0)
		visited[key] = false
	}

	if repo != nil {
		if err := repo.UpdateRoutes(ri.indexToAsset, ri.routesMap); err != nil {
			return err
		}
	}

	// sort all found routes by their length and the token index of the provided pairs
	for _, routeMap := range ri.routesMap {
		for _, route := range routeMap {
			sort.Slice(route, func(i, j int) bool {
				if len(route[i]) != len(route[j]) {
					return len(route[i]) < len(route[j])
				}
				lmt := len(route[i])
				for idx := 0; idx < lmt; idx++ {
					if route[i][idx] == route[j][idx] {
						continue
					}
					return route[i][idx] < route[j][idx]
				}
				return true
			})
		}
	}

	return nil
}

// findAllRoutes finds all routes from the given start index
// it runs recursively until it reaches the maxHopCount
func (ri *routeInfoImpl) findAllRoutes(start, current int, route []int, visited map[int]bool, hopCount uint) {
	if hopCount > ri.maxHopCount {
		return
	}

	nodes := ri.initialPairMap[current]

	for to := range nodes {
		if v, ok := visited[to]; ok && v {
			continue
		}
		if ri.routesMap[start] == nil {
			ri.routesMap[start] = make(map[int][][]int)
		}
		if ri.routesMap[start][to] == nil {
			ri.routesMap[start][to] = make([][]int, 0)
		}

		copiedRoute := make([]int, len(route))
		copy(copiedRoute, route)
		newRoute := append(copiedRoute, to)
		routes := ri.routesMap[start][to]
		ri.routesMap[start][to] = append(routes, newRoute)
		visited[to] = true
		ri.findAllRoutes(start, to, newRoute, visited, hopCount+1)
		visited[to] = false
	}
}
