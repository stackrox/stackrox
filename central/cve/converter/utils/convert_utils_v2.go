package utils

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/features"
)

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

	// Determine the CVSS score version based on available CVSS data.
	// Only set a specific version (V2 or V3) if the score is greater than 0.
	// A score of 0 indicates that the CVSS score is not available for this CVE,
	// in which case we should return UNKNOWN to avoid confusion.
	var scoreVersion storage.CVEInfo_ScoreVersion
	var impactScore float32
	scoreVersion = storage.CVEInfo_UNKNOWN // Default to UNKNOWN
	if from.GetCvssV3() != nil && from.GetCvssV3().GetScore() > 0 {
		scoreVersion = storage.CVEInfo_V3
		impactScore = from.GetCvssV3().GetImpactScore()
	} else if from.GetCvssV2() != nil && from.GetCvssV2().GetScore() > 0 {
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
