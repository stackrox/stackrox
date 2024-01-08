package memlimit

import (
	"os"
	"runtime/debug"
	"strconv"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/size"
)

const (
	goMemLimit  = `GOMEMLIMIT`
	roxMemLimit = `ROX_MEMLIMIT`
)

var (
	log = logging.LoggerForModule()

	// setMemoryLimit specifies the function to use when setting the (soft) memory limit.
	//
	// This exists for testing purposes.
	setMemoryLimit = debug.SetMemoryLimit
)

// SetMemoryLimit sets a (soft) memory limit on the Go runtime.
// See debug.SetMemoryLimit for more information.
func SetMemoryLimit() {
	if goLimit := os.Getenv(goMemLimit); goLimit != "" {
		log.Infof("%s set to %s", goMemLimit, goLimit)
		return
	}

	if roxLimit := os.Getenv(roxMemLimit); roxLimit != "" {
		l, err := strconv.ParseInt(roxLimit, 10, 64)
		if err != nil {
			log.Errorf("%s (%s) must be an integer in bytes: %v", roxMemLimit, roxLimit, err)
			return
		}
		// Set limit to 95% of the maximum.
		l -= l / 20
		setMemoryLimit(l)
		log.Infof("%s set to %.2fGi", roxMemLimit, float64(l)/size.GB)
		return
	}
	log.Warnf("Neither %s nor %s set", goMemLimit, roxMemLimit)
}
