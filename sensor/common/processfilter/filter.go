package processfilter

import (
	"math"

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
		fanOutLevels, fanOutLevelsWarning := env.ProcessFilterFanOutLevels.IntegerArraySetting()

		if fanOutLevelsWarning != "" {
			log.Warn(fanOutLevelsWarning)
		}
		// Set the maximum number of paths to the max integer in order to not filter out new processes in Sensor
		singletonFilter = filter.NewFilter(maxExactPathMatches, math.MaxInt, fanOutLevels)
	})
	return singletonFilter
}
