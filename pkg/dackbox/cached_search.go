package dackbox

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/dbhelper"
)

type cachedSearcher struct {
	graph           graph.RGraph
	predicate       func([]byte) (bool, error)
	searchPathElems []*dbhelper.BucketHandler
	nextKeys        func(*dbhelper.BucketHandler, dbhelper.Graph, []byte) [][]byte

	caches []map[string]bool
}

// NewCachedSearcher creates a new cached searcher that allows to efficiently perform multiple searches
// in a depth-first manner along a predefined path.
// Searches are performed via the `Search` method, where, starting with an unprefixed ID, the
// object graph is traversed along the specified bucket path to check a node that satisfies the
// given predicate (only the last node on the path is checked against the predicate).
// The search results will be cached not only for the final but also for the nodes visited along
// the path.
func NewCachedSearcher(g graph.RGraph, predicate func([]byte) (bool, error), searchPath BucketPath) Searcher {
	if g == nil || searchPath.Len() == 0 || predicate == nil {
		panic(errors.New("cannot search in a nil graph, along an empty path or with a nil predicate"))
	}

	nextKeys := (*dbhelper.BucketHandler).GetFilteredRefsFrom
	if searchPath.BackwardTraversal {
		nextKeys = (*dbhelper.BucketHandler).GetFilteredRefsTo
	}

	// We create a single map for each depth layer during a search, since the key spaces between layers are
	// disjoint. This is slightly more efficient since each rehashing will need to move a smaller amount of data.
	caches := make([]map[string]bool, 0, searchPath.Len())
	for i := 0; i < searchPath.Len(); i++ {
		caches = append(caches, make(map[string]bool))
	}

	return &cachedSearcher{
		graph:           g,
		predicate:       predicate,
		searchPathElems: searchPath.Elements,
		nextKeys:        nextKeys,
		caches:          caches,
	}
}

// Search performs a search.
func (c *cachedSearcher) Search(_ context.Context, unprefixedID string) (bool, error) {
	key := c.searchPathElems[0].GetKey(unprefixedID)

	return c.dfs(key, c.searchPathElems[1:])
}

func (c *cachedSearcher) dfs(current []byte, restPath []*dbhelper.BucketHandler) (bool, error) {
	currStr := string(current)
	if cachedResult, ok := c.caches[len(restPath)][currStr]; ok {
		return cachedResult, nil
	}

	var err error
	result := false
	if len(restPath) == 0 {
		result, err = c.predicate(current)
		if err != nil {
			return false, err
		}
	} else {
		for _, nextKey := range c.nextKeys(restPath[0], c.graph, current) {
			result, err = c.dfs(nextKey, restPath[1:])
			if err != nil {
				return false, err
			}
			if result {
				break
			}
		}
	}
	c.caches[len(restPath)][currStr] = result
	return result, nil
}

// NewCachedBucketReachabilityChecker returns a cachedSearcher that checks if there is a path of object-level references
// that follows the given bucket search path.
func NewCachedBucketReachabilityChecker(g graph.RGraph, searchPath BucketPath) Searcher {
	if g == nil || searchPath.Len() < 2 {
		panic(errors.New("cannot check bucket reachability in a nil graph, or along path of length shorter than 2"))
	}

	lastElem := searchPath.Elements[searchPath.Len()-1]
	shortenedPath := BucketPath{
		Elements:          searchPath.Elements[:searchPath.Len()-1],
		BackwardTraversal: searchPath.BackwardTraversal,
	}

	var pred func([]byte) (bool, error)
	if searchPath.BackwardTraversal {
		pred = func(key []byte) (bool, error) {
			return lastElem.HasFilteredRefsTo(g, key), nil
		}
	} else {
		pred = func(key []byte) (bool, error) {
			return lastElem.HasFilteredRefsFrom(g, key), nil
		}
	}

	return NewCachedSearcher(g, pred, shortenedPath)
}
