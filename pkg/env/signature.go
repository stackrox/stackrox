package env

import "time"

var (
	// DisableSignatureFetching disables signature fetching within the reprocessing loop.
	DisableSignatureFetching = RegisterBooleanSetting("ROX_DISABLE_SIGNATURE_FETCHING", false)

	// RedHatSigningKeyBundleURL is the remote URL of the key bundle JSON.
	// If empty, the key bundle updater does not start.
	RedHatSigningKeyBundleURL = RegisterSetting("ROX_REDHAT_SIGNING_KEY_BUNDLE_URL", AllowEmpty())

	// RedHatSigningKeyBundlePath is the local file path where the key bundle is read from.
	RedHatSigningKeyBundlePath = RegisterSetting("ROX_REDHAT_SIGNING_KEY_BUNDLE_PATH",
		WithDefault("/tmp/redhat-signing-keys/bundle.json"))

	// RedHatSigningKeyUpdateInterval controls how often the updater re-downloads the bundle.
	RedHatSigningKeyUpdateInterval = registerDurationSetting("ROX_REDHAT_SIGNING_KEY_UPDATE_INTERVAL", 4*time.Hour)

	// RedHatSigningKeyWatchInterval controls how often the watcher polls the file for changes.
	RedHatSigningKeyWatchInterval = registerDurationSetting("ROX_REDHAT_SIGNING_KEY_WATCH_INTERVAL", 30*time.Second)
)
