package features

// AttemptManifestDigest enables attempting to pull manifest digests from registres that historically did not
// support it but now appear to (ie: Nexus and RHEL).
var AttemptManifestDigest = registerFeature("Enables attempts to pull manifest digests for all registry integrations", "ROX_ATTEMPT_MANIFEST_DIGEST", enabled)
