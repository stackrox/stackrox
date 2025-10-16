package utils

import (
	"strings"
	"time"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cve"
	pkgCVSSV2 "github.com/stackrox/rox/pkg/cvss/cvssv2"
	pkgCVSSV3 "github.com/stackrox/rox/pkg/cvss/cvssv3"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/scans"
)

const (
	timeFormat = "2006-01-02T15:04Z"
)

// CVEType is the type of a CVE fetched by fetcher
type CVEType int32

// K8s is type for k8s CVEs, Istio is type for istio CVEs, OpenShift is type from OpenShift CVEs.
const (
	K8s = iota
	Istio
	OpenShift
)

func (c CVEType) String() string {
	switch c {
	case K8s:
		return "Kubernetes"
	case Istio:
		return "Istio"
	case OpenShift:
		return "OpenShift"
	}
	return "Unknown"
}

// ToStorageCVEType convert a CVEType to its corresponding storage CVE type.
func (c CVEType) ToStorageCVEType() storage.CVE_CVEType {
	switch c {
	case K8s:
		return storage.CVE_K8S_CVE
	case Istio:
		return storage.CVE_ISTIO_CVE
	case OpenShift:
		return storage.CVE_OPENSHIFT_CVE
	}
	return storage.CVE_UNKNOWN_CVE
}

// NVDCVEToEmbeddedCVE converts a *schema.NVDCVEFeedJSON10DefCVEItem to *storage.EmbeddedVulnerability.
func NVDCVEToEmbeddedCVE(nvdCVE *schema.NVDCVEFeedJSON10DefCVEItem, ct CVEType) (*storage.EmbeddedVulnerability, error) {
	if nvdCVE == nil || nvdCVE.CVE == nil || nvdCVE.CVE.CVEDataMeta == nil {
		return nil, errors.Errorf("Missing CVE or CVE MetaData for type: %d", ct)
	}

	if nvdCVE.Impact == nil || (nvdCVE.Impact.BaseMetricV2 == nil && nvdCVE.Impact.BaseMetricV3 == nil) {
		return nil, errors.New("CVE does not have either a CVSSv2 nor a CVSSv3 score")
	}

	cve := &storage.EmbeddedVulnerability{}
	cve.SetCve(nvdCVE.CVE.CVEDataMeta.ID)

	switch ct {
	case K8s:
		cve.SetVulnerabilityType(storage.EmbeddedVulnerability_K8S_VULNERABILITY)
	case Istio:
		cve.SetVulnerabilityType(storage.EmbeddedVulnerability_ISTIO_VULNERABILITY)
	case OpenShift:
		cve.SetVulnerabilityType(storage.EmbeddedVulnerability_OPENSHIFT_VULNERABILITY)
	default:
		return nil, errors.Errorf("unknown CVE type: %d", ct)
	}

	if nvdCVE.Impact.BaseMetricV2 != nil {
		cvssv2, err := nvdCvssv2ToProtoCvssv2(nvdCVE.Impact.BaseMetricV2)
		if err != nil {
			return nil, err
		}
		cve.SetCvssV2(cvssv2)
		cve.SetCvss(cvssv2.GetScore())
		cve.SetScoreVersion(storage.EmbeddedVulnerability_V2)
	}

	// If CVSSv3 is specified, prefer it over CVSSv2.
	if nvdCVE.Impact.BaseMetricV3 != nil {
		cvssv3, err := nvdCvssv3ToProtoCvssv3(nvdCVE.Impact.BaseMetricV3)
		if err != nil {
			return nil, err
		}
		cve.SetCvssV3(cvssv3)
		cve.SetCvss(cvssv3.GetScore())
		cve.SetScoreVersion(storage.EmbeddedVulnerability_V3)
	}

	if nvdCVE.PublishedDate != "" {
		if ts, err := time.Parse(timeFormat, nvdCVE.PublishedDate); err == nil {
			cve.SetPublishedOn(protoconv.ConvertTimeToTimestamp(ts))
		}
	}

	if nvdCVE.LastModifiedDate != "" {
		if ts, err := time.Parse(timeFormat, nvdCVE.LastModifiedDate); err == nil {
			cve.SetLastModified(protoconv.ConvertTimeToTimestamp(ts))
		}
	}

	// We have already checked that "nvdCVE.CVE" is not nil.
	if nvdCVE.CVE.Description != nil && len(nvdCVE.CVE.Description.DescriptionData) > 0 && nvdCVE.CVE.Description.DescriptionData[0] != nil {
		cve.SetSummary(nvdCVE.CVE.Description.DescriptionData[0].Value)
	}

	fixedByVersions := GetFixedVersions(nvdCVE)
	if len(fixedByVersions) > 0 {
		cve.Set_FixedBy(strings.Join(fixedByVersions, ","))
	}

	cve.SetLink(scans.GetVulnLink(cve.GetCve()))
	return cve, nil
}

func nvdCvssv2ToProtoCvssv2(baseMetricV2 *schema.NVDCVEFeedJSON10DefImpactBaseMetricV2) (*storage.CVSSV2, error) {
	if baseMetricV2 == nil || baseMetricV2.CVSSV2 == nil {
		return nil, errors.New("Missing CVSS Version 2 data")
	}

	cvssV2, err := pkgCVSSV2.ParseCVSSV2(baseMetricV2.CVSSV2.VectorString)
	if err != nil {
		return nil, err
	}

	if baseMetricV2.Severity != "" {
		k := strings.ToUpper(baseMetricV2.Severity[:1])
		sv, err := pkgCVSSV2.GetSeverityMapProtoVal(k)
		if err != nil {
			return nil, err
		}
		cvssV2.SetSeverity(sv)
	}

	cvssV2.SetScore(float32(baseMetricV2.CVSSV2.BaseScore))
	cvssV2.SetExploitabilityScore(float32(baseMetricV2.ExploitabilityScore))
	cvssV2.SetImpactScore(float32(baseMetricV2.ImpactScore))

	return cvssV2, nil
}

func nvdCvssv3ToProtoCvssv3(baseMetricV3 *schema.NVDCVEFeedJSON10DefImpactBaseMetricV3) (*storage.CVSSV3, error) {
	if baseMetricV3 == nil || baseMetricV3.CVSSV3 == nil {
		return nil, errors.New("Missing CVSS Version 3 data")
	}

	cvssV3, err := pkgCVSSV3.ParseCVSSV3(baseMetricV3.CVSSV3.VectorString)
	if err != nil {
		return nil, err
	}
	if baseMetricV3.CVSSV3.BaseSeverity != "" {
		k := strings.ToUpper(baseMetricV3.CVSSV3.BaseSeverity[:1])
		sv, err := pkgCVSSV3.GetSeverityMapProtoVal(k)
		if err != nil {
			return nil, err
		}
		cvssV3.SetSeverity(sv)
	}

	cvssV3.SetScore(float32(baseMetricV3.CVSSV3.BaseScore))
	cvssV3.SetExploitabilityScore(float32(baseMetricV3.ExploitabilityScore))
	cvssV3.SetImpactScore(float32(baseMetricV3.ImpactScore))

	return cvssV3, nil
}

// NVDCVEsToEmbeddedCVEs converts *schema.NVDCVEFeedJSON10DefCVEItem CVEs to *storage.EmbeddedVulnerability objects.
func NVDCVEsToEmbeddedCVEs(cves []*schema.NVDCVEFeedJSON10DefCVEItem, ct CVEType) ([]*storage.EmbeddedVulnerability, error) {
	ret := make([]*storage.EmbeddedVulnerability, 0, len(cves))
	for _, cve := range cves {
		ev, err := NVDCVEToEmbeddedCVE(cve, ct)
		if err != nil {
			return nil, err
		}
		ret = append(ret, ev)
	}
	return ret, nil
}

// Deprecated: replaced with equivalent functions using storage.ImageCVEV2
// ImageCVEToEmbeddedVulnerability coverts a Proto CVEs to Embedded Vuln
// It converts all the fields except Fixed By which gets set depending on the CVE
// TODO(ROX-28123): Remove
func ImageCVEToEmbeddedVulnerability(vuln *storage.ImageCVE) *storage.EmbeddedVulnerability {
	embeddedCVE := &storage.EmbeddedVulnerability{}
	embeddedCVE.SetCve(vuln.GetCveBaseInfo().GetCve())
	embeddedCVE.SetCvss(vuln.GetCvss())
	embeddedCVE.SetSummary(vuln.GetCveBaseInfo().GetSummary())
	embeddedCVE.SetLink(vuln.GetCveBaseInfo().GetLink())
	embeddedCVE.SetCvssV2(vuln.GetCveBaseInfo().GetCvssV2())
	embeddedCVE.SetCvssV3(vuln.GetCveBaseInfo().GetCvssV3())
	embeddedCVE.SetPublishedOn(vuln.GetCveBaseInfo().GetPublishedOn())
	embeddedCVE.SetLastModified(vuln.GetCveBaseInfo().GetLastModified())
	embeddedCVE.SetFirstSystemOccurrence(vuln.GetCveBaseInfo().GetCreatedAt())
	embeddedCVE.SetSuppressed(vuln.GetSnoozed())
	embeddedCVE.SetSuppressActivation(vuln.GetSnoozeStart())
	embeddedCVE.SetSuppressExpiry(vuln.GetSnoozeExpiry())
	embeddedCVE.SetSeverity(vuln.GetSeverity())
	embeddedCVE.SetCvssMetrics(vuln.GetCvssMetrics())
	embeddedCVE.SetNvdCvss(vuln.GetNvdcvss())
	embeddedCVE.SetEpss(vuln.GetCveBaseInfo().GetEpss())
	if vuln.GetCveBaseInfo().GetCvssV3() != nil {
		embeddedCVE.SetScoreVersion(storage.EmbeddedVulnerability_V3)
	} else {
		embeddedCVE.SetScoreVersion(storage.EmbeddedVulnerability_V2)
	}
	embeddedCVE.SetVulnerabilityType(storage.EmbeddedVulnerability_IMAGE_VULNERABILITY)
	embeddedCVE.SetVulnerabilityTypes([]storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY})
	return embeddedCVE
}

// NodeCVEToNodeVulnerability coverts a Proto CVEs to Embedded node vulnerability.
// It converts all the fields except fields that depend on the node context.
func NodeCVEToNodeVulnerability(protoCVE *storage.NodeCVE) *storage.NodeVulnerability {
	embeddedCVE := &storage.NodeVulnerability{}
	embeddedCVE.SetCveBaseInfo(protoCVE.GetCveBaseInfo())
	embeddedCVE.SetSeverity(protoCVE.GetSeverity())
	embeddedCVE.SetCvss(protoCVE.GetCvss())
	embeddedCVE.SetSnoozed(protoCVE.GetSnoozed())
	embeddedCVE.SetSnoozeStart(protoCVE.GetSnoozeStart())
	embeddedCVE.SetSnoozeExpiry(protoCVE.GetSnoozeExpiry())
	return embeddedCVE
}

// EmbeddedCVEToProtoCVE converts *storage.EmbeddedVulnerability object to *storage.CVE object
func EmbeddedCVEToProtoCVE(os string, from *storage.EmbeddedVulnerability) *storage.CVE {
	ret := &storage.CVE{}
	ret.SetType(embeddedVulnTypeToProtoType(from.GetVulnerabilityType()))
	ret.SetId(cve.ID(from.GetCve(), os))
	ret.SetCvss(from.GetCvss())
	ret.SetSummary(from.GetSummary())
	ret.SetLink(from.GetLink())
	ret.SetPublishedOn(from.GetPublishedOn())
	ret.SetLastModified(from.GetLastModified())
	ret.SetCvssV2(from.GetCvssV2())
	ret.SetCvssV3(from.GetCvssV3())
	ret.SetSuppressed(from.GetSuppressed())
	ret.SetSuppressActivation(from.GetSuppressActivation())
	ret.SetSuppressExpiry(from.GetSuppressExpiry())
	if ret.GetCvssV3() != nil {
		ret.SetScoreVersion(storage.CVE_V3)
		ret.SetImpactScore(from.GetCvssV3().GetImpactScore())
	} else if ret.GetCvssV2() != nil {
		ret.SetScoreVersion(storage.CVE_V2)
		ret.SetImpactScore(from.GetCvssV2().GetImpactScore())
	}

	ret.SetSeverity(from.GetSeverity())
	return ret
}

// Deprecated: replaced with equivalent functions using storage.ImageCVEV2
// EmbeddedVulnerabilityToImageCVE converts *storage.EmbeddedVulnerability object to *storage.ImageCVE object
// TODO(ROX-28123): Remove
func EmbeddedVulnerabilityToImageCVE(os string, from *storage.EmbeddedVulnerability) *storage.ImageCVE {
	var nvdCvss float32
	nvdCvss = 0
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

	cVEInfo := &storage.CVEInfo{}
	cVEInfo.SetCve(from.GetCve())
	cVEInfo.SetSummary(from.GetSummary())
	cVEInfo.SetLink(from.GetLink())
	cVEInfo.SetPublishedOn(from.GetPublishedOn())
	cVEInfo.SetCreatedAt(from.GetFirstSystemOccurrence())
	cVEInfo.SetLastModified(from.GetLastModified())
	cVEInfo.SetCvssV2(from.GetCvssV2())
	cVEInfo.SetCvssV3(from.GetCvssV3())
	cVEInfo.SetEpss(from.GetEpss())
	ret := &storage.ImageCVE{}
	ret.SetId(cve.ID(from.GetCve(), os))
	ret.SetOperatingSystem(os)
	ret.SetCveBaseInfo(cVEInfo)
	ret.SetCvss(from.GetCvss())
	ret.SetNvdcvss(nvdCvss)
	ret.SetNvdScoreVersion(nvdVersion)
	ret.SetSeverity(from.GetSeverity())
	ret.SetSnoozed(from.GetSuppressed())
	ret.SetSnoozeStart(from.GetSuppressActivation())
	ret.SetSnoozeExpiry(from.GetSuppressExpiry())
	ret.SetCvssMetrics(from.GetCvssMetrics())
	if ret.GetCveBaseInfo().GetCvssV3() != nil {
		ret.GetCveBaseInfo().SetScoreVersion(storage.CVEInfo_V3)
		ret.SetImpactScore(from.GetCvssV3().GetImpactScore())
	} else if ret.GetCveBaseInfo().GetCvssV2() != nil {
		ret.GetCveBaseInfo().SetScoreVersion(storage.CVEInfo_V2)
		ret.SetImpactScore(from.GetCvssV2().GetImpactScore())
	}
	return ret
}

// EmbeddedVulnerabilityToClusterCVE converts *storage.EmbeddedVulnerability object to *storage.ClusterCVE object
func EmbeddedVulnerabilityToClusterCVE(cveType storage.CVE_CVEType, from *storage.EmbeddedVulnerability) *storage.ClusterCVE {
	cVEInfo := &storage.CVEInfo{}
	cVEInfo.SetCve(from.GetCve())
	cVEInfo.SetSummary(from.GetSummary())
	cVEInfo.SetLink(from.GetLink())
	cVEInfo.SetPublishedOn(from.GetPublishedOn())
	cVEInfo.SetCreatedAt(from.GetFirstSystemOccurrence())
	cVEInfo.SetLastModified(from.GetLastModified())
	cVEInfo.SetCvssV2(from.GetCvssV2())
	cVEInfo.SetCvssV3(from.GetCvssV3())
	ret := &storage.ClusterCVE{}
	ret.SetId(cve.ID(from.GetCve(), cveType.String()))
	ret.SetCveBaseInfo(cVEInfo)
	ret.SetCvss(from.GetCvss())
	ret.SetSeverity(from.GetSeverity())
	ret.SetSnoozed(from.GetSuppressed())
	ret.SetSnoozeStart(from.GetSuppressActivation())
	ret.SetSnoozeExpiry(from.GetSuppressExpiry())
	ret.SetType(cveType)
	if ret.GetCveBaseInfo().GetCvssV3() != nil {
		ret.GetCveBaseInfo().SetScoreVersion(storage.CVEInfo_V3)
		ret.SetImpactScore(from.GetCvssV3().GetImpactScore())
	} else if ret.GetCveBaseInfo().GetCvssV2() != nil {
		ret.GetCveBaseInfo().SetScoreVersion(storage.CVEInfo_V2)
		ret.SetImpactScore(from.GetCvssV2().GetImpactScore())
	}
	return ret
}

// NodeVulnerabilityToNodeCVE converts *storage.NodeVulnerability object to *storage.NodeCVE object
func NodeVulnerabilityToNodeCVE(os string, from *storage.NodeVulnerability) *storage.NodeCVE {
	ret := &storage.NodeCVE{}
	ret.SetId(cve.ID(from.GetCveBaseInfo().GetCve(), os))
	ret.SetCveBaseInfo(from.GetCveBaseInfo())
	ret.SetCvss(from.GetCvss())
	ret.SetOperatingSystem(os)
	ret.SetSeverity(from.GetSeverity())
	ret.SetSnoozed(from.GetSnoozed())
	ret.SetSnoozeStart(from.GetSnoozeStart())
	ret.SetSnoozeExpiry(from.GetSnoozeExpiry())
	if from.GetCveBaseInfo().GetCvssV3() != nil {
		ret.SetImpactScore(from.GetCveBaseInfo().GetCvssV3().GetImpactScore())
	} else if from.GetCveBaseInfo().GetCvssV2() != nil {
		ret.SetImpactScore(from.GetCveBaseInfo().GetCvssV2().GetImpactScore())
	}
	return ret
}

func embeddedVulnTypeToProtoType(protoCVEType storage.EmbeddedVulnerability_VulnerabilityType) storage.CVE_CVEType {
	switch protoCVEType {
	case storage.EmbeddedVulnerability_IMAGE_VULNERABILITY:
		return storage.CVE_IMAGE_CVE
	case storage.EmbeddedVulnerability_K8S_VULNERABILITY:
		return storage.CVE_K8S_CVE
	case storage.EmbeddedVulnerability_ISTIO_VULNERABILITY:
		return storage.CVE_ISTIO_CVE
	case storage.EmbeddedVulnerability_NODE_VULNERABILITY:
		return storage.CVE_NODE_CVE
	case storage.EmbeddedVulnerability_OPENSHIFT_VULNERABILITY:
		return storage.CVE_OPENSHIFT_CVE
	default:
		return storage.CVE_UNKNOWN_CVE
	}
}

// GetFixedVersions gets the fixed version from a NVD CVE item.
func GetFixedVersions(nvdCVE *schema.NVDCVEFeedJSON10DefCVEItem) []string {
	var versions []string
	if nvdCVE == nil || nvdCVE.Configurations == nil {
		return versions
	}

	for _, node := range nvdCVE.Configurations.Nodes {
		if node == nil {
			continue
		}

		for _, cpeMatch := range node.CPEMatch {
			if cpeMatch != nil && cpeMatch.VersionEndExcluding != "" {
				versions = append(versions, cpeMatch.VersionEndExcluding)
			}
		}
	}

	return versions
}
