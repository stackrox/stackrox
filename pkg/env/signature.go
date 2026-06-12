package env

import "time"

var (
	// DisableSignatureFetching disables signature fetching within the reprocessing loop.
	DisableSignatureFetching = RegisterBooleanSetting("ROX_DISABLE_SIGNATURE_FETCHING", false)

	// RedHatSigningKeyWatchInterval controls how often the watcher polls the key bundle file.
	// Set to 0 to disable the watcher entirely.
	RedHatSigningKeyWatchInterval = registerDurationSetting("ROX_REDHAT_SIGNING_KEY_WATCH_INTERVAL", 4*time.Hour, WithDurationZeroAllowed())
)
