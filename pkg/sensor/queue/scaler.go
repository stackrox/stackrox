package queue

import (
	"math"
	"os"
	"strconv"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log             = logging.LoggerForModule()
	defaultMemlimit = float64(4194304000)
)

// ScaleSize will scale the size of a given queue size based on the Sensor memory limit relative
// to the default memory limit of 4GB. It returns the scaled queue size variable, which is at least 1.
func ScaleSize(queueSize int) (int, error) {
	if roxLimit := os.Getenv("ROX_MEMLIMIT"); roxLimit != "" {
		l, err := strconv.ParseInt(roxLimit, 10, 64)
		if err != nil {
			log.Errorf("ROX_MEMLIMIT must be an integer in bytes: %v", err)
			return -1, err
		}
		if l == 0 {
			log.Warn("ROX_MEMLIMIT is set to 0!")
		}
		ratio := float64(l) / defaultMemlimit // FIXME: Convert correctly

		log.Warnf("Got effective memlimit of %d. Scaling queue to %.2f percent", l, ratio*100) // FIXME: Remove

		queueSize = int(math.Round(ratio * float64(queueSize)))
		if queueSize <= 0 {
			// Ensure that we always have at least a queue size of 1
			queueSize = 1
		}
	}
	return queueSize, nil
}

// ScaleSizeOnNonDefault only scales the given integer setting if it is
// set to its default value (e.g. not changed by hand).
func ScaleSizeOnNonDefault(setting *env.IntegerSetting) int {
	v := setting.IntegerSetting()
	if v != setting.DefaultValue() {
		// Setting has been changed somewhere - don't scale it
		log.Debugf("Detected non-default value. Not scaling %s", setting.EnvVar())
		return v
	}

	scaled, err := ScaleSize(v)
	if err != nil {
		log.Warnf("Failed to scale setting %s. Returning its unscaled value", setting.EnvVar())
		return v
	}

	log.Infof("Scaling %s - Default: %d, Scaled: %d", setting.EnvVar(), v, scaled)
	return scaled
}
