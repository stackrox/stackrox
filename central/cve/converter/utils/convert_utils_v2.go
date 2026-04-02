package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/uuid"
)

// cveV1Namespace is a stable UUID namespace for deterministic NormalizedCVE IDs.
var cveV1Namespace = uuid.NewV5FromNonUUIDs("stackrox", "normalized-cve-v1")

// DeterministicCVEID derives a stable UUID for a NormalizedCVE from its content_hash.
// The same content_hash always produces the same UUID, making ON CONFLICT(id) DO UPDATE idempotent.
// A new content_hash produces a new UUID, inserting a new row.
func DeterministicCVEID(contentHash string) string {
	return uuid.NewV5(cveV1Namespace, contentHash).String()
}

// ImageCVEV2ToEmbeddedVulnerability coverts a `*storage.ImageCVEV2` to an `*storage.EmbeddedVulnerability`.
func ImageCVEV2ToEmbeddedVulnerability(vuln *storage.ImageCVEV2) *storage.EmbeddedVulnerability {
	scoreVersion := storage.EmbeddedVulnerability_V2
	if vuln.GetCveBaseInfo().GetCvssV3() != nil {
		scoreVersion = storage.EmbeddedVulnerability_V3
	}

	ret := &storage.EmbeddedVulnerability{
		Cve:                   vuln.GetCveBaseInfo().GetCve(),
		Cvss:                  vuln.GetCvss(),
		Summary:               vuln.GetCveBaseInfo().GetSummary(),
		Link:                  vuln.GetCveBaseInfo().GetLink(),
		ScoreVersion:          scoreVersion,
		CvssV2:                vuln.GetCveBaseInfo().GetCvssV2(),
		CvssV3:                vuln.GetCveBaseInfo().GetCvssV3(),
		PublishedOn:           vuln.GetCveBaseInfo().GetPublishedOn(),
		LastModified:          vuln.GetCveBaseInfo().GetLastModified(),
		FirstSystemOccurrence: vuln.GetCveBaseInfo().GetCreatedAt(),
		FixAvailableTimestamp: vuln.GetFixAvailableTimestamp(),
		Severity:              vuln.GetSeverity(),
		CvssMetrics:           vuln.GetCveBaseInfo().GetCvssMetrics(),
		NvdCvss:               vuln.GetNvdcvss(),
		Epss:                  vuln.GetCveBaseInfo().GetEpss(),
		FirstImageOccurrence:  vuln.GetFirstImageOccurrence(),
		VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
		State:                 vuln.GetState(),
		Datasource:            vuln.GetDatasource(),
	}

	if vuln.GetIsFixable() {
		ret.SetFixedBy = &storage.EmbeddedVulnerability_FixedBy{
			FixedBy: vuln.GetFixedBy(),
		}
	}

	if vuln.GetAdvisory() != nil {
		ret.Advisory = &storage.Advisory{
			Name: vuln.GetAdvisory().GetName(),
			Link: vuln.GetAdvisory().GetLink(),
		}
	}

	return ret
}

// EmbeddedVulnerabilityToImageCVEV2 converts *storage.EmbeddedVulnerability object to *storage.ImageCVEV2 object
func EmbeddedVulnerabilityToImageCVEV2(imageID string, componentID string, index int, from *storage.EmbeddedVulnerability) (*storage.ImageCVEV2, error) {
	var nvdCvss float32
	nvdVersion := storage.CvssScoreVersion_UNKNOWN_VERSION
	for _, score := range from.GetCvssMetrics() {
		if score.GetSource() == storage.Source_SOURCE_NVD {
			if score.GetCvssv3() != nil {
				nvdCvss = score.GetCvssv3().GetScore()
				nvdVersion = storage.CvssScoreVersion_V3
			} else if score.GetCvssv2() != nil {
				nvdCvss = score.GetCvssv2().GetScore()
				nvdVersion = storage.CvssScoreVersion_V2
			}
		}
	}

	var scoreVersion storage.CVEInfo_ScoreVersion
	var impactScore float32
	if from.GetCvssV3() != nil {
		scoreVersion = storage.CVEInfo_V3
		impactScore = from.GetCvssV3().GetImpactScore()
	} else if from.GetCvssV2() != nil {
		scoreVersion = storage.CVEInfo_V2
		impactScore = from.GetCvssV2().GetImpactScore()
	}

	cveID := cve.IDV2(from, componentID, index)

	ret := &storage.ImageCVEV2{
		Id:          cveID,
		ComponentId: componentID,
		CveBaseInfo: &storage.CVEInfo{
			Cve:          from.GetCve(),
			Summary:      from.GetSummary(),
			Link:         from.GetLink(),
			PublishedOn:  from.GetPublishedOn(),
			CreatedAt:    from.GetFirstSystemOccurrence(),
			LastModified: from.GetLastModified(),
			CvssV2:       from.GetCvssV2(),
			CvssV3:       from.GetCvssV3(),
			CvssMetrics:  from.GetCvssMetrics(),
			Epss:         from.GetEpss(),
			ScoreVersion: scoreVersion,
		},
		Cvss:                  from.GetCvss(),
		Nvdcvss:               nvdCvss,
		NvdScoreVersion:       nvdVersion,
		Severity:              from.GetSeverity(),
		FirstImageOccurrence:  from.GetFirstImageOccurrence(),
		FixAvailableTimestamp: from.GetFixAvailableTimestamp(),
		State:                 from.GetState(),
		IsFixable:             from.GetFixedBy() != "",
		ImpactScore:           impactScore,
		Advisory:              from.GetAdvisory(),
		Datasource:            from.GetDatasource(),
	}
	if !features.FlattenImageData.Enabled() {
		ret.ImageId = imageID
	} else {
		ret.ImageIdV2 = imageID
	}

	if from.GetFixedBy() != "" {
		ret.HasFixedBy = &storage.ImageCVEV2_FixedBy{
			FixedBy: from.GetFixedBy(),
		}
	}

	return ret, nil
}

// ComputeCVEContentHash computes a SHA256 hash of the CVE content fields using
// null-byte delimiters to prevent boundary-collision attacks.
// The cvss_v3 score is formatted to 2 decimal places (e.g., "7.50"), or empty string if zero.
// All other fields are UTF-8 strings.
func ComputeCVEContentHash(cveName, source, severity string, cvssV3 float32, summary string) string {
	var cvssV3Str string
	if cvssV3 == 0.0 {
		cvssV3Str = ""
	} else {
		cvssV3Str = fmt.Sprintf("%.2f", cvssV3)
	}

	input := cveName + "\x00" + source + "\x00" + severity + "\x00" + cvssV3Str + "\x00" + summary
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// SourceToString converts a storage.Source enum to the canonical string used in
// the CVE content hash and the cves.source column.
func SourceToString(source storage.Source) string {
	switch source {
	case storage.Source_SOURCE_OSV:
		return "OSV"
	case storage.Source_SOURCE_NVD:
		return "NVD"
	case storage.Source_SOURCE_RED_HAT:
		return "RED_HAT"
	default:
		return "UNKNOWN"
	}
}

// SeverityToString converts a storage.VulnerabilitySeverity enum to the canonical
// string used in the CVE content hash and the cves.severity column.
func SeverityToString(severity storage.VulnerabilitySeverity) string {
	switch severity {
	case storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY:
		return "CRITICAL"
	case storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY:
		return "HIGH"
	case storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY:
		return "MEDIUM"
	case storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY:
		return "LOW"
	default:
		return "UNKNOWN"
	}
}

// SeverityFromString converts the canonical severity string stored in the cves
// table back to the storage.VulnerabilitySeverity enum.
func SeverityFromString(s string) storage.VulnerabilitySeverity {
	switch s {
	case "CRITICAL":
		return storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
	case "HIGH":
		return storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY
	case "MEDIUM":
		return storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY
	case "LOW":
		return storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY
	default:
		return storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	}
}
