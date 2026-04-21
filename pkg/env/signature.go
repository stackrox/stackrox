package env

import "time"

var (
	// DisableSignatureFetching disables signature fetching within the reprocessing loop.
	DisableSignatureFetching = RegisterBooleanSetting("ROX_DISABLE_SIGNATURE_FETCHING", false)

	// RedHatSigningKeyManifestURL is the URL of the manifest listing trusted Red Hat signing keys.
	RedHatSigningKeyManifestURL = RegisterSetting("ROX_REDHAT_SIGNING_KEY_MANIFEST_URL")

	// RedHatSigningKeyUpdateInterval controls how often the signing key updater runs.
	RedHatSigningKeyUpdateInterval = registerDurationSetting("ROX_REDHAT_SIGNING_KEY_UPDATE_INTERVAL", 4*time.Hour)

	// RedHatSigningKeysRuntimeDir is the writable directory where downloaded Red Hat signing keys are stored.
	RedHatSigningKeysRuntimeDir = RegisterSetting("ROX_REDHAT_SIGNING_KEYS_RUNTIME_DIR",
		WithDefault("/var/lib/stackrox/signature-keys/redhat"))
)
