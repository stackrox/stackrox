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

	ret := &storage.EmbeddedVulnerability{}
	ret.SetCve(vuln.GetCveBaseInfo().GetCve())
	ret.SetCvss(vuln.GetCvss())
	ret.SetSummary(vuln.GetCveBaseInfo().GetSummary())
	ret.SetLink(vuln.GetCveBaseInfo().GetLink())
	ret.SetScoreVersion(scoreVersion)
	ret.SetCvssV2(vuln.GetCveBaseInfo().GetCvssV2())
	ret.SetCvssV3(vuln.GetCveBaseInfo().GetCvssV3())
	ret.SetPublishedOn(vuln.GetCveBaseInfo().GetPublishedOn())
	ret.SetLastModified(vuln.GetCveBaseInfo().GetLastModified())
	ret.SetFirstSystemOccurrence(vuln.GetCveBaseInfo().GetCreatedAt())
	ret.SetSeverity(vuln.GetSeverity())
	ret.SetCvssMetrics(vuln.GetCveBaseInfo().GetCvssMetrics())
	ret.SetNvdCvss(vuln.GetNvdcvss())
	ret.SetEpss(vuln.GetCveBaseInfo().GetEpss())
	ret.SetFirstImageOccurrence(vuln.GetFirstImageOccurrence())
	ret.SetVulnerabilityType(storage.EmbeddedVulnerability_IMAGE_VULNERABILITY)
	ret.SetVulnerabilityTypes([]storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY})
	ret.SetState(vuln.GetState())

	if vuln.GetIsFixable() {
		ret.Set_FixedBy(vuln.GetFixedBy())
	}

	if vuln.GetAdvisory() != nil {
		advisory := &storage.Advisory{}
		advisory.SetName(vuln.GetAdvisory().GetName())
		advisory.SetLink(vuln.GetAdvisory().GetLink())
		ret.SetAdvisory(advisory)
	}

	return ret
}

// EmbeddedVulnerabilityToImageCVEV2 converts *storage.EmbeddedVulnerability object to *storage.ImageCVEV2 object
func EmbeddedVulnerabilityToImageCVEV2(imageID string, componentID string, from *storage.EmbeddedVulnerability) (*storage.ImageCVEV2, error) {
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

	cveID, err := cve.IDV2(from, componentID)
	if err != nil {
		return nil, err
	}

	cVEInfo := &storage.CVEInfo{}
	cVEInfo.SetCve(from.GetCve())
	cVEInfo.SetSummary(from.GetSummary())
	cVEInfo.SetLink(from.GetLink())
	cVEInfo.SetPublishedOn(from.GetPublishedOn())
	cVEInfo.SetCreatedAt(from.GetFirstSystemOccurrence())
	cVEInfo.SetLastModified(from.GetLastModified())
	cVEInfo.SetCvssV2(from.GetCvssV2())
	cVEInfo.SetCvssV3(from.GetCvssV3())
	cVEInfo.SetCvssMetrics(from.GetCvssMetrics())
	cVEInfo.SetEpss(from.GetEpss())
	cVEInfo.SetScoreVersion(scoreVersion)
	ret := &storage.ImageCVEV2{}
	ret.SetId(cveID)
	ret.SetComponentId(componentID)
	ret.SetCveBaseInfo(cVEInfo)
	ret.SetCvss(from.GetCvss())
	ret.SetNvdcvss(nvdCvss)
	ret.SetNvdScoreVersion(nvdVersion)
	ret.SetSeverity(from.GetSeverity())
	ret.SetFirstImageOccurrence(from.GetFirstImageOccurrence())
	ret.SetState(from.GetState())
	ret.SetIsFixable(from.GetFixedBy() != "")
	ret.SetImpactScore(impactScore)
	ret.SetAdvisory(from.GetAdvisory())
	if !features.FlattenImageData.Enabled() {
		ret.SetImageId(imageID)
	} else {
		ret.SetImageIdV2(imageID)
	}

	if from.GetFixedBy() != "" {
		ret.SetFixedBy(from.GetFixedBy())
	}

	return ret, nil
}
