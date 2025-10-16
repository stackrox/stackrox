package converter

import (
	"github.com/stackrox/rox/generated/storage"
)

// FillV2NodeVulnerabilities populates the Vulnerabilities node scan component field from the Vulns one.
func FillV2NodeVulnerabilities(node *storage.Node) {
	for _, component := range node.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			nodeVuln := EmbeddedVulnerabilityToNodeVulnerability(vuln)
			component.Vulnerabilities = append(component.Vulnerabilities, nodeVuln)
		}
	}
}

// MoveNodeVulnsToNewField populates the Vulnerabilities (new) node scan component field from the Vulns (legacy) and clears Vulns field.
func MoveNodeVulnsToNewField(node *storage.Node) {
	FillV2NodeVulnerabilities(node)
	for _, component := range node.GetScan().GetComponents() {
		component.Vulns = nil
	}
}

// EmbeddedVulnerabilityToNodeVulnerability converts a *storage.EmbeddedVulnerability object to a *storage.NodeVulnerability one.
func EmbeddedVulnerabilityToNodeVulnerability(vuln *storage.EmbeddedVulnerability) *storage.NodeVulnerability {
	ret := &storage.NodeVulnerability{
		CveBaseInfo: &storage.CVEInfo{
			Cve:          vuln.GetCve(),
			Summary:      vuln.GetSummary(),
			Link:         vuln.GetLink(),
			PublishedOn:  vuln.GetPublishedOn(),
			CreatedAt:    vuln.GetFirstSystemOccurrence(),
			LastModified: vuln.GetLastModified(),
			CvssV3:       vuln.GetCvssV3(),
			CvssV2:       vuln.GetCvssV2(),
			ScoreVersion: cveInfoScoreVersion(vuln.GetScoreVersion()),
		},
		Cvss:         vuln.GetCvss(),
		Severity:     vuln.GetSeverity(),
		Snoozed:      vuln.GetSuppressed(),
		SnoozeStart:  vuln.GetSuppressActivation(),
		SnoozeExpiry: vuln.GetSuppressExpiry(),
	}
	if vuln.GetSetFixedBy() != nil {
		ret.SetFixedBy = &storage.NodeVulnerability_FixedBy{
			FixedBy: vuln.GetFixedBy(),
		}
	}
	return ret
}

func cveInfoScoreVersion(scoreVersion storage.EmbeddedVulnerability_ScoreVersion) storage.CVEInfo_ScoreVersion {
	switch scoreVersion {
	case storage.EmbeddedVulnerability_V3:
		return storage.CVEInfo_V3
	case storage.EmbeddedVulnerability_V2:
		return storage.CVEInfo_V2
	default:
		return storage.CVEInfo_UNKNOWN
	}
}
