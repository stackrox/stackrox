package env

import "time"

var (
	// DisableSignatureFetching disables signature fetching within the reprocessing loop.
	DisableSignatureFetching = RegisterBooleanSetting("ROX_DISABLE_SIGNATURE_FETCHING", false)

	// RedHatSigningKeyBundlePath is the local file path where the key bundle is read from.
	RedHatSigningKeyBundlePath = RegisterSetting("ROX_REDHAT_SIGNING_KEY_BUNDLE_PATH",
		WithDefault("/tmp/redhat-signing-keys/bundle.json"))

	// RedHatSigningKeyWatchInterval controls how often the watcher polls the file for changes.
	RedHatSigningKeyWatchInterval = registerDurationSetting("ROX_REDHAT_SIGNING_KEY_WATCH_INTERVAL", 30*time.Second)
)
