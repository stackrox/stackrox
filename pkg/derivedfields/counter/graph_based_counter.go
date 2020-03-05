package counter

import (
	"context"

	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/dackbox/sortedkeys"
	"github.com/stackrox/rox/pkg/search/filtered"
	"github.com/stackrox/rox/pkg/set"
)

// NewGraphBasedDerivedFieldCounter generates derived field count for input keys by traversing the RGraph
func NewGraphBasedDerivedFieldCounter(graphProvider graph.Provider, dackboxPath dackbox.Path, sacFilters ...filtered.Filter) DerivedFieldCounter {
	return &graphBasedDerivedFieldCounterImpl{
		forwardTraversal: dackboxPath.ForwardTraversal,
		graphProvider:    graphProvider,
		prefixPath:       dackboxPath.Path,
		sacFilters:       sacFilters,
	}
}

type graphBasedDerivedFieldCounterImpl struct {
	forwardTraversal bool
	graphProvider    graph.Provider
	prefixPath       [][]byte
	sacFilters       []filtered.Filter
}

func (c *graphBasedDerivedFieldCounterImpl) Count(ctx context.Context, keys ...string) (map[string]int32, error) {
	// prefix the initial set of keys, since they will be prefixed in the graph.
	currentIDs := make([][]byte, 0, len(keys))
	for _, key := range keys {
		currentIDs = append(currentIDs, badgerhelper.GetBucketKey(c.prefixPath[0], []byte(key)))
	}

	idGraph := c.graphProvider.NewGraphView()
	defer idGraph.Discard()

	var step func([]byte) [][]byte
	if c.forwardTraversal {
		step = idGraph.GetRefsFrom
	} else {
		step = idGraph.GetRefsTo
	}

	return count(ctx, currentIDs, c.prefixPath, step, c.sacFilters)
}

func count(ctx context.Context, currentIDs [][]byte, prefixPath [][]byte, step func([]byte) [][]byte, sacFilter []filtered.Filter) (map[string]int32, error) {
	counts := make(map[string]int32)
	cache := make(map[string]int32)
	var err error
	for _, currentID := range currentIDs {
		counts[GetIDForKey(prefixPath[0], currentID)], err = dfs(ctx, currentID, cache, set.NewStringSet(), prefixPath, step, sacFilter)
		if err != nil {
			return nil, err
		}
	}
	return counts, nil
}

func dfs(ctx context.Context, currentID []byte, cache map[string]int32, seenIDs set.StringSet, prefixPath [][]byte, step func([]byte) [][]byte, sacFilters []filtered.Filter) (int32, error) {
	if len(prefixPath) == 0 {
		return 0, nil
	}

	if seenIDs.Contains(string(currentID)) {
		return 0, nil
	}
	seenIDs.Add(string(currentID))

	if count, ok := cache[string(currentID)]; ok {
		return count, nil
	}

	// Destination prefix visited
	if len(prefixPath) == 1 {
		id := GetIDForKey(prefixPath[0], currentID)
		// Perform SAC check only on final prefix
		allowed, err := filtered.ApplySACFilters(ctx, []string{id}, sacFilters...)
		if err != nil || len(allowed) == 0 {
			return 0, err
		}
		return 1, nil
	}

	// Cannot reach destination
	transformedIDs := step(currentID)
	if len(transformedIDs) == 0 {
		return 0, nil
	}

	totalCount := int32(0)
	nextIDs := filterByPrefix(prefixPath[1], transformedIDs)
	for _, nextID := range nextIDs {
		count, err := dfs(ctx, nextID, cache, seenIDs, prefixPath[1:], step, sacFilters)
		if err != nil {
			return 0, err
		}
		totalCount += count
	}
	cache[string(currentID)] = totalCount
	return totalCount, nil
}

func filterByPrefix(prefix []byte, input sortedkeys.SortedKeys) sortedkeys.SortedKeys {
	filteredKeys := input[:0]
	for _, key := range input {
		if badgerhelper.HasPrefix(prefix, key) {
			filteredKeys = append(filteredKeys, key)
		}
	}
	return filteredKeys
}

// GetIDForKey returns id for a prefixed key
func GetIDForKey(prefix, key []byte) string {
	return string(badgerhelper.StripBucket(prefix, key))
}
