package env

import "time"

var (
	// DisableSignatureFetching disables signature fetching within the reprocessing loop.
	DisableSignatureFetching = RegisterBooleanSetting("ROX_DISABLE_SIGNATURE_FETCHING", false)

	// DisableRedHatSigningKeyBundleWatcher disables the file-based watcher that polls
	// for Red Hat signing key bundle updates. Use as a kill switch if the watcher
	// causes issues in production.
	DisableRedHatSigningKeyBundleWatcher = RegisterBooleanSetting("ROX_DISABLE_REDHAT_SIGNING_KEY_BUNDLE_WATCHER", false)

	// RedHatSigningKeyBundlePath is the local file path where the key bundle is read from.
	RedHatSigningKeyBundlePath = RegisterSetting("ROX_REDHAT_SIGNING_KEY_BUNDLE_PATH",
		WithDefault("/tmp/redhat-signing-keys/bundle.json"))

	// RedHatSigningKeyWatchInterval controls how often the watcher polls the file for changes.
	RedHatSigningKeyWatchInterval = registerDurationSetting("ROX_REDHAT_SIGNING_KEY_WATCH_INTERVAL", 4*time.Hour)
)
