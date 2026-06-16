package env

import "time"

var (
	// DisableSignatureFetching disables signature fetching within the reprocessing loop.
	DisableSignatureFetching = RegisterBooleanSetting("ROX_DISABLE_SIGNATURE_FETCHING", false)

	// RedHatSigningKeyBundleURL is the remote URL of the key bundle JSON.
	// If empty, the key bundle updater does not start.
	RedHatSigningKeyBundleURL = RegisterSetting("ROX_REDHAT_SIGNING_KEY_BUNDLE_URL", AllowEmpty())

	// RedHatSigningKeyBundleFilePath is the local file path where the key bundle is stored.
	// The updater writes downloaded bundles here; the watcher polls it for changes.
	// In offline mode, users can mount a bundle at this path for manual key updates.
	RedHatSigningKeyBundleFilePath = RegisterSetting("ROX_REDHAT_SIGNING_KEY_BUNDLE_FILE_PATH",
		WithDefault("/run/stackrox.io/redhat-signing-keys/bundle.json"))

	// RedHatSigningKeyUpdateInterval controls how often the updater re-downloads the bundle.
	RedHatSigningKeyUpdateInterval = registerDurationSetting("ROX_REDHAT_SIGNING_KEY_UPDATE_INTERVAL", 4*time.Hour)

	// RedHatSigningKeyWatchInterval controls how often the watcher polls the key bundle file.
	// Set to 0 to disable the watcher entirely.
	RedHatSigningKeyWatchInterval = registerDurationSetting("ROX_REDHAT_SIGNING_KEY_WATCH_INTERVAL", 4*time.Hour, WithDurationZeroAllowed())
)
