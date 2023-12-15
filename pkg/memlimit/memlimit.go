package memlimit

import (
	"os"
	"runtime/debug"
	"strconv"

	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

func SetMemoryLimit() {
	var limit int64
	defer func() {
		if limit == 0 {
			log.Warn("Memory limit left unset")
			return
		}

		limitGi := float64(limit) / 1024 / 1024 / 1024
		log.Infof("Memory limit set: %.2fGi", limitGi)
	}()

	var err error
	if goLimit := os.Getenv(`GOMEMLIMIT`); goLimit != "" {
		if limit, err = strconv.ParseInt(goLimit, 10, 64); err == nil {
			return
		}
	}

	if roxLimit := os.Getenv(`ROX_MEMLIMIT`); roxLimit != "" {
		limit, err = strconv.ParseInt(roxLimit, 10, 64)
		if err == nil {
			// Set limit to 95% of the maximum.
			limit -= limit / 20
			debug.SetMemoryLimit(limit)
		}
	}

	limit = setMemoryLimit()
}
