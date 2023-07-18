package processfilter

import (
	"math"

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
		// Set the maximum number of paths to the max integer in order to not filter out new processes in Sensor
		singletonFilter = filter.NewFilter(maxExactPathMatches, math.MaxInt, bucketSizes)
	})
	return singletonFilter
}
