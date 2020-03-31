package filtered

import (
	"github.com/stackrox/rox/pkg/dackbox/sortedkeys"
	"github.com/stackrox/rox/pkg/dbhelper"
)

func filterByPrefix(prefix []byte, input sortedkeys.SortedKeys) sortedkeys.SortedKeys {
	filtered := make([][]byte, 0, len(input))
	for _, key := range input {
		if dbhelper.HasPrefix(prefix, key) {
			filtered = append(filtered, key)
		}
	}
	return filtered
}
