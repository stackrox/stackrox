package cvss

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
)

// VulnI provides functionality to get vulnerability score.
type VulnI interface {
	GetSeverity() storage.VulnerabilitySeverity
	GetCvssV2() *storage.CVSSV2
	GetCvssV3() *storage.CVSSV3
	GetScoreVersion() storage.CVEInfo_ScoreVersion
}

// NewFromEmbeddedVulnerability returns an instance of VulnI for *storage.EmbeddedVulnerability.
func NewFromEmbeddedVulnerability(vuln *storage.EmbeddedVulnerability) VulnI {
	return &vulnScoreInfo{
		severity:     vuln.GetSeverity(),
		cvssV3:       vuln.GetCvssV3(),
		cvssv2:       vuln.GetCvssV2(),
		scoreVersion: scoreVersionFromEmbeddedVuln(vuln),
	}
}

// NewFromCVE returns an instance of VulnI for *storage.CVE.
func NewFromCVE(vuln *storage.CVE) VulnI {
	return &vulnScoreInfo{
		severity:     vuln.GetSeverity(),
		cvssV3:       vuln.GetCvssV3(),
		cvssv2:       vuln.GetCvssV2(),
		scoreVersion: scoreVersionFromCVE(vuln),
	}
}

// NewFromNodeVulnerability returns an instance of VulnI for *storage.NodeVulnerability.
func NewFromNodeVulnerability(vuln *storage.NodeVulnerability) VulnI {
	return &vulnScoreInfo{
		severity:     vuln.GetSeverity(),
		cvssV3:       vuln.GetCveBaseInfo().GetCvssV3(),
		cvssv2:       vuln.GetCveBaseInfo().GetCvssV2(),
		scoreVersion: vuln.GetCveBaseInfo().GetScoreVersion(),
	}
}

type vulnScoreInfo struct {
	severity     storage.VulnerabilitySeverity
	cvssv2       *storage.CVSSV2
	cvssV3       *storage.CVSSV3
	scoreVersion storage.CVEInfo_ScoreVersion
}

func (v *vulnScoreInfo) GetSeverity() storage.VulnerabilitySeverity {
	return v.severity
}

func (v *vulnScoreInfo) GetCvssV2() *storage.CVSSV2 {
	return v.cvssv2
}

func (v *vulnScoreInfo) GetCvssV3() *storage.CVSSV3 {
	return v.cvssV3
}

func (v *vulnScoreInfo) GetScoreVersion() storage.CVEInfo_ScoreVersion {
	return v.scoreVersion
}

// VulnToSeverity to returns a storage severity
func VulnToSeverity(v VulnI) storage.VulnerabilitySeverity {
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

// FormatSeverity converts the given storage.VulnerabilitySeverity to a more human-readable string.
// ex: LOW_VULNERABILITY_SEVERITY -> Low
func FormatSeverity(severity storage.VulnerabilitySeverity) string {
	return strings.Title(strings.ToLower(strings.TrimSuffix(severity.String(), "_VULNERABILITY_SEVERITY")))
}

func scoreVersionFromEmbeddedVuln(vuln *storage.EmbeddedVulnerability) storage.CVEInfo_ScoreVersion {
	switch vuln.GetScoreVersion() {
	case storage.EmbeddedVulnerability_V3:
		return storage.CVEInfo_V3
	case storage.EmbeddedVulnerability_V2:
		return storage.CVEInfo_V2
	default:
		return storage.CVEInfo_UNKNOWN
	}
}

func scoreVersionFromCVE(vuln *storage.CVE) storage.CVEInfo_ScoreVersion {
	switch vuln.GetScoreVersion() {
	case storage.CVE_V3:
		return storage.CVEInfo_V3
	case storage.CVE_V2:
		return storage.CVEInfo_V2
	default:
		return storage.CVEInfo_UNKNOWN
	}
}
