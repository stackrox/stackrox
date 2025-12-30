package filter

import (
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	singletonInstance sync.Once

	maxExactPathMatches = env.ProcessFilterMaxExactPathMatches.IntegerSetting()
	maxUniqueProcesses  = env.ProcessFilterMaxProcessPaths.IntegerSetting()
	fanOutLevels        = env.ProcessFilterFanOutLevels.IntegerArraySetting()

	singletonFilter filter.Filter
)

// Singleton returns a global, threadsafe process filter
func Singleton() filter.Filter {
	singletonInstance.Do(func() {
		singletonFilter = filter.NewFilter(maxExactPathMatches, maxUniqueProcesses, fanOutLevels)
	})
	return singletonFilter
}
