package filtered

import (
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/dackbox/sortedkeys"
)

func filterByPrefix(prefix []byte, input sortedkeys.SortedKeys) sortedkeys.SortedKeys {
	filtered := make([][]byte, 0, len(input))
	for _, key := range input {
		if badgerhelper.HasPrefix(prefix, key) {
			filtered = append(filtered, key)
		}
	}
	return filtered
}
