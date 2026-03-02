package env

import "time"

var (
	// RedHatSigningKeyUpdateInterval defines the interval at which Red Hat's signing key (used in the default Red Hat
	// signature integration) is updated.
	RedHatSigningKeyUpdateInterval = registerDurationSetting("ROX_REDHAT_SIGNING_KEY_UPDATE_INTERVAL", 4*time.Hour)

	// DisableSignatureFetching disables signature fetching within the reprocessing loop.
	DisableSignatureFetching = RegisterBooleanSetting("ROX_DISABLE_SIGNATURE_FETCHING", false)
)
