package env

import (
	"time"
)

var (
	// WhitelistGenerationDuration will set the duration for which a new whitelist remains unlocked
	WhitelistGenerationDuration = registerDurationSetting("ROX_WHITELIST_GENERATION_DURATION", time.Hour)
)
