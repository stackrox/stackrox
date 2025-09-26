package enricher

// import (
// 	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
// 	"github.com/stackrox/rox/generated/storage"
// )

// These functions are copied/adapted from node enricher utilities
// func normalizedSeverity(severity v4.VulnerabilityReport_Vulnerability_Severity) storage.VulnerabilitySeverity {
// 	switch severity {
// 	case v4.VulnerabilityReport_Vulnerability_SEVERITY_LOW:
// 		return storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY
// 	case v4.VulnerabilityReport_Vulnerability_SEVERITY_MODERATE:
// 		return storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY
// 	case v4.VulnerabilityReport_Vulnerability_SEVERITY_IMPORTANT:
// 		return storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY
// 	case v4.VulnerabilityReport_Vulnerability_SEVERITY_CRITICAL:
// 		return storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
// 	default:
// 		return storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
// 	}
// }

// func maybeOverwriteSeverity(vuln *storage.EmbeddedVulnerability) {
// 	// Overwrite severity based on CVSS scores if available
// 	// This logic is adapted from scanner v4 node conversion
// 	if vuln.GetCvss() == 0 {
// 		return
// 	}

// 	score := vuln.GetCvss()
// 	var newSeverity storage.VulnerabilitySeverity

// 	switch {
// 	case score >= 9.0:
// 		newSeverity = storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
// 	case score >= 7.0:
// 		newSeverity = storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY
// 	case score >= 4.0:
// 		newSeverity = storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY
// 	case score >= 0.1:
// 		newSeverity = storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY
// 	default:
// 		newSeverity = storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
// 	}

// 	// Only overwrite if the new severity is more severe than the current one
// 	if shouldOverwriteSeverity(vuln.GetSeverity(), newSeverity) {
// 		vuln.Severity = newSeverity
// 	}
// }

// func shouldOverwriteSeverity(current, proposed storage.VulnerabilitySeverity) bool {
// 	severityOrder := map[storage.VulnerabilitySeverity]int{
// 		storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY:   0,
// 		storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY:       1,
// 		storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY:  2,
// 		storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY: 3,
// 		storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY:  4,
// 	}

// 	currentOrder, currentExists := severityOrder[current]
// 	proposedOrder, proposedExists := severityOrder[proposed]

// 	if !currentExists || !proposedExists {
// 		return false
// 	}

// 	return proposedOrder > currentOrder
// }
