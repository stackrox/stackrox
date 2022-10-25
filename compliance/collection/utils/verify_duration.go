package utils

import (
	"time"

	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// VerifyAndUpdateDuration ensures that a given duration is positive bigger than zero and returns a default otherwise
func VerifyAndUpdateDuration(dur, defaultDur time.Duration) time.Duration {
	if (dur) <= 0 {
		log.Warn("Negative or zero duration found. Setting to 4 hours.")
		return defaultDur
	}
	return dur
}
