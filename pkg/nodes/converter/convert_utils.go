package converter

import (
	"github.com/stackrox/rox/generated/storage"
)

// FillV2NodeVulnerabilities populates the Vulnerabilities node scan component field from the Vulns one.
func FillV2NodeVulnerabilities(node *storage.Node) {
	for _, component := range node.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			nodeVuln := EmbeddedVulnerabilityToNodeVulnerability(vuln)
			component.SetVulnerabilities(append(component.GetVulnerabilities(), nodeVuln))
		}
	}
}

// MoveNodeVulnsToNewField populates the Vulnerabilities (new) node scan component field from the Vulns (legacy) and clears Vulns field.
func MoveNodeVulnsToNewField(node *storage.Node) {
	FillV2NodeVulnerabilities(node)
	for _, component := range node.GetScan().GetComponents() {
		component.SetVulns(nil)
	}
}

// EmbeddedVulnerabilityToNodeVulnerability converts a *storage.EmbeddedVulnerability object to a *storage.NodeVulnerability one.
func EmbeddedVulnerabilityToNodeVulnerability(vuln *storage.EmbeddedVulnerability) *storage.NodeVulnerability {
	cVEInfo := &storage.CVEInfo{}
	cVEInfo.SetCve(vuln.GetCve())
	cVEInfo.SetSummary(vuln.GetSummary())
	cVEInfo.SetLink(vuln.GetLink())
	cVEInfo.SetPublishedOn(vuln.GetPublishedOn())
	cVEInfo.SetCreatedAt(vuln.GetFirstSystemOccurrence())
	cVEInfo.SetLastModified(vuln.GetLastModified())
	cVEInfo.SetCvssV3(vuln.GetCvssV3())
	cVEInfo.SetCvssV2(vuln.GetCvssV2())
	cVEInfo.SetScoreVersion(cveInfoScoreVersion(vuln.GetScoreVersion()))
	ret := &storage.NodeVulnerability{}
	ret.SetCveBaseInfo(cVEInfo)
	ret.SetCvss(vuln.GetCvss())
	ret.SetSeverity(vuln.GetSeverity())
	ret.SetSnoozed(vuln.GetSuppressed())
	ret.SetSnoozeStart(vuln.GetSuppressActivation())
	ret.SetSnoozeExpiry(vuln.GetSuppressExpiry())
	if vuln.GetSetFixedBy() != nil {
		ret.Set_FixedBy(vuln.GetFixedBy())
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
