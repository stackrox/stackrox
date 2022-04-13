package counter

import (
	"context"

	"github.com/stackrox/stackrox/pkg/dackbox"
	"github.com/stackrox/stackrox/pkg/dackbox/graph"
	"github.com/stackrox/stackrox/pkg/dbhelper"
	"github.com/stackrox/stackrox/pkg/search/filtered"
	"github.com/stackrox/stackrox/pkg/set"
)

// NewGraphBasedDerivedFieldCounter generates derived field count for input keys by traversing the RGraph
func NewGraphBasedDerivedFieldCounter(graphProvider graph.Provider, dackboxPath dackbox.Path, sacFilter filtered.Filter) DerivedFieldCounter {
	return &graphBasedDerivedFieldCounterImpl{
		forwardTraversal: dackboxPath.ForwardTraversal,
		graphProvider:    graphProvider,
		prefixPath:       dackboxPath.Path,
		sacFilter:        sacFilter,
	}
}

type graphBasedDerivedFieldCounterImpl struct {
	forwardTraversal bool
	graphProvider    graph.Provider
	prefixPath       [][]byte
	sacFilter        filtered.Filter
}

func (c *graphBasedDerivedFieldCounterImpl) Count(ctx context.Context, keys ...string) (map[string]int32, error) {
	// prefix the initial set of keys, since they will be prefixed in the graph.
	currentIDs := make([][]byte, 0, len(keys))
	for _, key := range keys {
		currentIDs = append(currentIDs, dbhelper.GetBucketKey(c.prefixPath[0], []byte(key)))
	}

	idGraph := c.graphProvider.NewGraphView()
	defer idGraph.Discard()

	var filteredStep func([]byte, []byte) [][]byte
	if c.forwardTraversal {
		filteredStep = idGraph.GetRefsFromPrefix
	} else {
		filteredStep = idGraph.GetRefsToPrefix
	}

	return count(ctx, currentIDs, c.prefixPath, filteredStep, c.sacFilter)
}

func count(ctx context.Context, currentIDs [][]byte, prefixPath [][]byte, filteredStep func([]byte, []byte) [][]byte, sacFilter filtered.Filter) (map[string]int32, error) {
	counts := make(map[string]int32)
	cache := make(map[string]int32)
	var err error
	for _, currentID := range currentIDs {
		counts[GetIDForKey(prefixPath[0], currentID)], err = dfs(ctx, currentID, cache, set.NewStringSet(), prefixPath, filteredStep, sacFilter)
		if err != nil {
			return nil, err
		}
	}
	return counts, nil
}

func dfs(ctx context.Context, currentID []byte, cache map[string]int32, seenIDs set.StringSet, prefixPath [][]byte, step func([]byte, []byte) [][]byte, sacFilter filtered.Filter) (int32, error) {
	if len(prefixPath) == 0 {
		return 0, nil
	}

	currentIDStr := string(currentID)
	if !seenIDs.Add(currentIDStr) {
		return 0, nil
	}

	if count, ok := cache[currentIDStr]; ok {
		return count, nil
	}

	count, err := func() (int32, error) {
		// Destination prefix visited
		if len(prefixPath) == 1 {
			id := GetIDForKey(prefixPath[0], currentID)
			// Perform SAC check only on final prefix
			allowed, err := filtered.ApplySACFilter(ctx, []string{id}, sacFilter)
			if err != nil || len(allowed) == 0 {
				return 0, err
			}
			return 1, nil
		}

		nextIDs := step(currentID, prefixPath[1])
		if len(nextIDs) == 0 {
			// Cannot reach destination
			return 0, nil
		}

		totalCount := int32(0)
		for _, nextID := range nextIDs {
			count, err := dfs(ctx, nextID, cache, seenIDs, prefixPath[1:], step, sacFilter)
			if err != nil {
				return 0, err
			}
			totalCount += count
		}
		return totalCount, nil
	}()
	if err != nil {
		return 0, err
	}
	cache[currentIDStr] = count
	return count, nil
}

// GetIDForKey returns id for a prefixed key
func GetIDForKey(prefix, key []byte) string {
	return string(dbhelper.StripBucket(prefix, key))
}
