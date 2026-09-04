package env

import "time"

var (
	// DisableSignatureFetching disables signature fetching within the reprocessing loop.
	DisableSignatureFetching = RegisterBooleanSetting("ROX_DISABLE_SIGNATURE_FETCHING", false)

	// RedHatSigningKeyBundleURL is the remote URL of the key bundle JSON.
	// If empty, the key bundle updater does not start.
	RedHatSigningKeyBundleURL = RegisterSetting("ROX_REDHAT_SIGNING_KEY_BUNDLE_URL",
		WithDefault("https://definitions.stackrox.io/signing-keys/bundle.json"), AllowEmpty())

	// RedHatSigningKeyUpdateInterval controls how often the updater re-downloads the bundle.
	RedHatSigningKeyUpdateInterval = registerDurationSetting("ROX_REDHAT_SIGNING_KEY_UPDATE_INTERVAL", 4*time.Hour)

	// RedHatSigningKeyWatchInterval controls how often the watcher polls the key bundle file.
	// Set to 0 to disable the watcher entirely.
	RedHatSigningKeyWatchInterval = registerDurationSetting("ROX_REDHAT_SIGNING_KEY_WATCH_INTERVAL", 4*time.Hour, WithDurationZeroAllowed())

	// ImageSignatureVerificationTimeout bounds how long the enricher waits for signature verification
	// of a single image against all configured signature integrations.
	// Keyless (cosign) verification performs remote RPCs (fetching the Sigstore TUF trust root and
	// Fulcio/Rekor material, and querying the transparency log), which can take several seconds,
	// especially on the first call before the TUF root is cached or when the Sigstore CDN is slow.
	// The default is generous enough to accommodate keyless verification while still bounding the
	// enrichment hot path.
	ImageSignatureVerificationTimeout = registerDurationSetting("ROX_IMAGE_SIGNATURE_VERIFICATION_TIMEOUT", 30*time.Second)
)
