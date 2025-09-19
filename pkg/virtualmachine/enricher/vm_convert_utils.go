package enricher

import (
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
)

// These functions are copied/adapted from node enricher utilities
func normalizedSeverity(severity v4.VulnerabilityReport_Vulnerability_Severity) storage.VulnerabilitySeverity {
	switch severity {
	case v4.VulnerabilityReport_Vulnerability_SEVERITY_LOW:
		return storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY
	case v4.VulnerabilityReport_Vulnerability_SEVERITY_MODERATE:
		return storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY
	case v4.VulnerabilityReport_Vulnerability_SEVERITY_IMPORTANT:
		return storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY
	case v4.VulnerabilityReport_Vulnerability_SEVERITY_CRITICAL:
		return storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
	default:
		return storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	}
}

func setScoresAndScoreVersions(vuln *storage.EmbeddedVulnerability, metrics []*v4.VulnerabilityReport_Vulnerability_CVSS) error {
	// Simplified CVSS scoring logic adapted from scanner v4 node conversion
	for _, metric := range metrics {
		if metric == nil {
			continue
		}

		// Set CVSS v2 scores
		if v2 := metric.GetV2(); v2 != nil {
			vuln.Cvss = v2.GetBaseScore()
			vuln.ScoreVersion = storage.EmbeddedVulnerability_V2
			if v2.GetVector() != "" {
				vuln.CvssV2 = &storage.CVSSV2{
					Vector: v2.GetVector(),
					Score:  v2.GetBaseScore(),
				}
			}
		}

		// Set CVSS v3 scores (prefer v3 over v2)
		if v3 := metric.GetV3(); v3 != nil {
			vuln.Cvss = v3.GetBaseScore()
			vuln.ScoreVersion = storage.EmbeddedVulnerability_V3
			if v3.GetVector() != "" {
				vuln.CvssV3 = &storage.CVSSV3{
					Vector: v3.GetVector(),
					Score:  v3.GetBaseScore(),
				}
			}
		}
	}
	return nil
}

func maybeOverwriteSeverity(vuln *storage.EmbeddedVulnerability) {
	// Overwrite severity based on CVSS scores if available
	// This logic is adapted from scanner v4 node conversion
	if vuln.GetCvss() == 0 {
		return
	}

	score := vuln.GetCvss()
	var newSeverity storage.VulnerabilitySeverity

	switch {
	case score >= 9.0:
		newSeverity = storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
	case score >= 7.0:
		newSeverity = storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY
	case score >= 4.0:
		newSeverity = storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY
	case score >= 0.1:
		newSeverity = storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY
	default:
		newSeverity = storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	}

	// Only overwrite if the new severity is more severe than the current one
	if vuln.GetSeverity() < newSeverity {
		vuln.Severity = newSeverity
	}
}
