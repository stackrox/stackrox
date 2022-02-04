package cvss

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
)

type vulnI interface {
	GetSeverity() storage.VulnerabilitySeverity
	GetCvssV2() *storage.CVSSV2
	GetCvssV3() *storage.CVSSV3
}

// VulnToSeverity to returns a storage severity
func VulnToSeverity(v vulnI) storage.VulnerabilitySeverity {
	if v.GetSeverity() != storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY {
		return v.GetSeverity()
	}

	if v.GetCvssV3() != nil {
		switch v.GetCvssV3().GetSeverity() {
		case storage.CVSSV3_UNKNOWN:
			return storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
		case storage.CVSSV3_NONE, storage.CVSSV3_LOW:
			return storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY
		case storage.CVSSV3_MEDIUM:
			return storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY
		case storage.CVSSV3_HIGH:
			return storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY
		case storage.CVSSV3_CRITICAL:
			return storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
		}
	}
	if v.GetCvssV2() != nil {
		switch v.GetCvssV2().GetSeverity() {
		case storage.CVSSV2_UNKNOWN:
			return storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
		case storage.CVSSV2_LOW:
			return storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY
		case storage.CVSSV2_MEDIUM:
			return storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY
		case storage.CVSSV2_HIGH:
			return storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY
		}
	}
	return storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
}

// StringToSeverity converts the given string representation of a severity into a storage.VulnerabilitySeverity.
func StringToSeverity(severity string) storage.VulnerabilitySeverity {
	switch strings.ToLower(severity) {
	case "low":
		return storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY
	case "moderate":
		return storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY
	case "important":
		return storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY
	case "critical":
		return storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
	default:
		return storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	}
}

// FormatSeverity converts the given storage.VulnerabilitySeverity to a more human-readable string.
// ex: LOW_VULNERABILITY_SEVERITY -> Low
func FormatSeverity(severity storage.VulnerabilitySeverity) string {
	return strings.Title(strings.ToLower(strings.TrimSuffix(severity.String(), "_VULNERABILITY_SEVERITY")))
}
