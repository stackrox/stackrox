package processfilter

import (
	"fmt"
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
		// Get effective configuration respecting both mode presets and individual overrides
		config, warnStr := env.GetEffectiveProcessFilterConfig()

		if warnStr != "" {
			log.Warn(warnStr)
		}

		modeStr := ""
		if modeConfig, _ := env.GetProcessFilterModeConfig(); modeConfig != nil {
			modeStr = fmt.Sprintf("mode=%s, ", env.ProcessFilterMode.Setting())
		}
		log.Infof("Process filter configuration: %smaxExactPathMatches=%d, fanOutLevels=%v",
			modeStr, config.MaxExactPathMatches, config.FanOutLevels)

		// Set the maximum number of paths to the max integer in order to not filter out new processes in Sensor
		singletonFilter = filter.NewFilter(config.MaxExactPathMatches, math.MaxInt, config.FanOutLevels)
	})
	return singletonFilter
}
