package filter

import (
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()

	singletonInstance sync.Once

	singletonFilter filter.Filter
)

// Singleton returns a global, threadsafe process filter
func Singleton() filter.Filter {
	singletonInstance.Do(func() {
		maxExactPathMatches := env.ProcessFilterMaxExactPathMatches.IntegerSetting()
		maxUniqueProcesses := env.ProcessFilterMaxProcessPaths.IntegerSetting()
		fanOutLevels, fanOutLevelsWarning := env.ProcessFilterFanOutLevels.IntegerArraySetting()

		if fanOutLevelsWarning != "" {
			log.Warn(fanOutLevelsWarning)
		}
		singletonFilter = filter.NewFilter(maxExactPathMatches, maxUniqueProcesses, fanOutLevels)
	})
	return singletonFilter
}
