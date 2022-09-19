package env

import (
	"time"
)

var (
	// BaselineGenerationDuration will set the duration for which a new excluded scope remains unlocked.
	BaselineGenerationDuration = registerDurationSetting("ROX_BASELINE_GENERATION_DURATION", time.Hour)
)
