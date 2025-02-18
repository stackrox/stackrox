package utils

import (
	"strconv"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cve"
)

// ImageCVEV2ToEmbeddedVulnerability coverts a Proto CVEs to Embedded Vuln
// It converts all the fields except Fixed By which gets set depending on the CVE
func ImageCVEV2ToEmbeddedVulnerability(vuln *storage.ImageCVEV2) *storage.EmbeddedVulnerability {
	var scoreVersion storage.EmbeddedVulnerability_ScoreVersion
	if vuln.GetCveBaseInfo().GetCvssV3() != nil {
		scoreVersion = storage.EmbeddedVulnerability_V3
	} else {
		scoreVersion = storage.EmbeddedVulnerability_V2
	}

	vulnType := storage.EmbeddedVulnerability_IMAGE_VULNERABILITY
	vulnTypes := []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY}

	ret := &storage.EmbeddedVulnerability{
		Cve:                   vuln.GetCveBaseInfo().GetCve(),
		Cvss:                  vuln.GetCvss(),
		Summary:               vuln.GetCveBaseInfo().GetSummary(),
		Link:                  vuln.GetCveBaseInfo().GetLink(),
		CvssV2:                vuln.GetCveBaseInfo().GetCvssV2(),
		CvssV3:                vuln.GetCveBaseInfo().GetCvssV3(),
		PublishedOn:           vuln.GetCveBaseInfo().GetPublishedOn(),
		LastModified:          vuln.GetCveBaseInfo().GetLastModified(),
		FirstSystemOccurrence: vuln.GetCveBaseInfo().GetCreatedAt(),
		Severity:              vuln.GetSeverity(),
		CvssMetrics:           vuln.GetCveBaseInfo().GetCvssMetrics(),
		NvdCvss:               vuln.GetNvdcvss(),
		Epss:                  vuln.GetCveBaseInfo().GetEpss(),
		FirstImageOccurrence:  vuln.GetFirstImageOccurrence(),
		ScoreVersion:          scoreVersion,
		VulnerabilityType:     vulnType,
		VulnerabilityTypes:    vulnTypes,
		State:                 vuln.GetState(),
	}

	if vuln.IsFixable {
		ret.SetFixedBy = &storage.EmbeddedVulnerability_FixedBy{
			FixedBy: vuln.GetFixedBy(),
		}
	}

	return ret
}

// EmbeddedVulnerabilityToImageCVEV2 converts *storage.EmbeddedVulnerability object to *storage.ImageCVEV2 object
func EmbeddedVulnerabilityToImageCVEV2(os string, imageID string, componentID string, cveIndex int, from *storage.EmbeddedVulnerability) *storage.ImageCVEV2 {
	var nvdCvss float32
	nvdCvss = 0
	nvdVersion := storage.CvssScoreVersion_UNKNOWN_VERSION
	for _, score := range from.GetCvssMetrics() {
		if score.Source == storage.Source_SOURCE_NVD {
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

	ret := &storage.ImageCVEV2{
		Id:              cve.IDV2(from.GetCve(), componentID, strconv.Itoa(cveIndex)),
		ComponentId:     componentID,
		OperatingSystem: os,
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
		Cvss:                 from.GetCvss(),
		Nvdcvss:              nvdCvss,
		NvdScoreVersion:      nvdVersion,
		Severity:             from.GetSeverity(),
		ImageId:              imageID,
		FirstImageOccurrence: from.GetFirstImageOccurrence(),
		State:                from.GetState(),
		IsFixable:            from.GetFixedBy() != "",
		ImpactScore:          impactScore,
	}

	if from.GetFixedBy() != "" {
		ret.HasFixedBy = &storage.ImageCVEV2_FixedBy{
			FixedBy: from.GetFixedBy(),
		}
	}

	return ret
}
