package memlimit

import (
	"fmt"
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
	var limit string
	var reason string

	defer func() {
		if limit == "" {
			log.Warnf("Memory limit left unset: %s", reason)
			return
		}

		prettyLimit := limit
		if l, err := strconv.ParseInt(limit, 10, 64); err == nil {
			limitGi := float64(l) / 1024 / 1024 / 1024
			prettyLimit = fmt.Sprintf("%.2fGi", limitGi)
		}


		log.Infof("Memory limit set: %s", prettyLimit)
	}()

	if goLimit := os.Getenv(`GOMEMLIMIT`); goLimit != "" {
		// Respect the set GOMEMLIMIT.
		limit = goLimit
		return
	}

	if roxLimit := os.Getenv(`ROX_MEMLIMIT`); roxLimit != "" {
		l, err := strconv.ParseInt(roxLimit, 10, 64)
		if err != nil {
			reason = fmt.Sprintf("ROX_MEMLIMIT set improperly (must be an integer in bytes): %v", err)
			return
		}

		// Set limit to 95% of the maximum.
		l -= l / 20
		setMemoryLimit(l)
		limit = strconv.FormatInt(l, 10)
	}

	reason = "Neither GOMEMLIMIT nor ROX_MEMLIMIT set"
}
