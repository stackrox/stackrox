package memlimit

import (
	"os"
	"runtime/debug"
	"strconv"

	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()

	// setMemoryLimit specifies the function to use when setting the (soft) memory limit.
	//
	// This exists for testing purposes.
	setMemoryLimit = debug.SetMemoryLimit
)

func SetMemoryLimit() {
	if goLimit := os.Getenv(`GOMEMLIMIT`); goLimit != "" {
		log.Infof("GOMEMLIMIT set to %s", goLimit)
		return
	}

	if roxLimit := os.Getenv(`ROX_MEMLIMIT`); roxLimit != "" {
		l, err := strconv.ParseInt(roxLimit, 10, 64)
		if err != nil {
			log.Errorf("ROX_MEMLIMIT (%s) must be an integer in bytes: %v", roxLimit, err)
			return
		}
		// Set limit to 95% of the maximum.
		l -= l / 20
		setMemoryLimit(l)
		log.Infof("ROX_MEMLIMIT set to %.2fGi", float64(l)/1024/1024/1024)
		return
	}
	log.Warn("Neither GOMEMLIMIT nor ROX_MEMLIMIT set")
}
