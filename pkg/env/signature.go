package env

import "time"

var (
	// RedHatSigningKeyBucketURL points to a plain-text file containing Red Hat's software signing public key
	RedHatSigningKeyBucketURL = RegisterSetting("ROX_REDHAT_SIGNING_KEY_BUCKET_URL", WithDefault("https://storage.googleapis.com/rox-public-key-test-20260203/rox-public-key-test-20260203.txt"))

	// RedHatSigningKeyUpdateInterval defines the interval at which Red Hat's signing key (used in the default Red Hat
	// signature integration) is updated.
	RedHatSigningKeyUpdateInterval = registerDurationSetting("ROX_REDHAT_SIGNING_KEY_UPDATE_INTERVAL", 1*time.Hour)

	// DisableSignatureFetching disables signature fetching within the reprocessing loop.
	DisableSignatureFetching = RegisterBooleanSetting("ROX_DISABLE_SIGNATURE_FETCHING", false)
)
