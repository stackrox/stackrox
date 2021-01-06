package env

import (
	"os"
	"time"
)

var (
	// BaselineGenerationDuration will set the duration for which a new excluded scope remains unlocked.
	//
	// TODO(ROX-6194): Initialize via calling `registerDurationSetting()` with
	//   the new env var after the deprecation cycle.
	BaselineGenerationDuration *DurationSetting
)

// TODO(ROX-6194): Remove this entirely after the deprecation cycle started with the 55.0 release.
func init() {
	legacyValue, legacyValueFound := os.LookupEnv("ROX_WHITELIST_GENERATION_DURATION")
	_, newValueFound := os.LookupEnv("ROX_BASELINE_GENERATION_DURATION")

	if !newValueFound && legacyValueFound {
		_ = os.Setenv("ROX_BASELINE_GENERATION_DURATION", legacyValue)
	}

	// Now we can pretend the "new" `ROX_BASELINE_GENERATION_DURATION` env var
	// is always present.
	BaselineGenerationDuration = registerDurationSetting("ROX_BASELINE_GENERATION_DURATION", time.Hour)
}
