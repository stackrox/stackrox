package filter

import (
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	maxExactPathMatches = 5
)

var (
	singletonInstance sync.Once

	bucketSizes     = []int{8, 6, 4, 2}
	singletonFilter filter.Filter
)

// Singleton returns a global, threadsafe process filter
func Singleton() filter.Filter {
	singletonInstance.Do(func() {
		singletonFilter = filter.NewFilter(maxExactPathMatches, bucketSizes)
	})
	return singletonFilter
}
