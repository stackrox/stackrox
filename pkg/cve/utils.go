package cve

import "github.com/stackrox/stackrox/generated/storage"

// IsCVESnoozed returns whether the cve is snoozed.
func IsCVESnoozed(cve *storage.EmbeddedVulnerability) bool {
	return cve.GetSuppressed() || cve.GetState() != storage.VulnerabilityState_OBSERVED
}
