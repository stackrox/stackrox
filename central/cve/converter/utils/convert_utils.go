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

	cve := &storage.EmbeddedVulnerability{
		Cve: nvdCVE.CVE.CVEDataMeta.ID,
	}

	switch ct {
	case K8s:
		cve.VulnerabilityType = storage.EmbeddedVulnerability_K8S_VULNERABILITY
	case Istio:
		cve.VulnerabilityType = storage.EmbeddedVulnerability_ISTIO_VULNERABILITY
	case OpenShift:
		cve.VulnerabilityType = storage.EmbeddedVulnerability_OPENSHIFT_VULNERABILITY
	default:
		return nil, errors.Errorf("unknown CVE type: %d", ct)
	}

	if nvdCVE.Impact.BaseMetricV2 != nil {
		cvssv2, err := nvdCvssv2ToProtoCvssv2(nvdCVE.Impact.BaseMetricV2)
		if err != nil {
			return nil, err
		}
		cve.CvssV2 = cvssv2
		cve.Cvss = cvssv2.Score
		cve.ScoreVersion = storage.EmbeddedVulnerability_V2
	}

	// If CVSSv3 is specified, prefer it over CVSSv2.
	if nvdCVE.Impact.BaseMetricV3 != nil {
		cvssv3, err := nvdCvssv3ToProtoCvssv3(nvdCVE.Impact.BaseMetricV3)
		if err != nil {
			return nil, err
		}
		cve.CvssV3 = cvssv3
		cve.Cvss = cvssv3.Score
		cve.ScoreVersion = storage.EmbeddedVulnerability_V3
	}

	if nvdCVE.PublishedDate != "" {
		if ts, err := time.Parse(timeFormat, nvdCVE.PublishedDate); err == nil {
			cve.PublishedOn = protoconv.ConvertTimeToTimestamp(ts)
		}
	}

	if nvdCVE.LastModifiedDate != "" {
		if ts, err := time.Parse(timeFormat, nvdCVE.LastModifiedDate); err == nil {
			cve.LastModified = protoconv.ConvertTimeToTimestamp(ts)
		}
	}

	// We have already checked that "nvdCVE.CVE" is not nil.
	if nvdCVE.CVE.Description != nil && len(nvdCVE.CVE.Description.DescriptionData) > 0 && nvdCVE.CVE.Description.DescriptionData[0] != nil {
		cve.Summary = nvdCVE.CVE.Description.DescriptionData[0].Value
	}

	fixedByVersions := GetFixedVersions(nvdCVE)
	if len(fixedByVersions) > 0 {
		cve.SetFixedBy = &storage.EmbeddedVulnerability_FixedBy{
			FixedBy: strings.Join(fixedByVersions, ","),
		}
	}

	cve.Link = scans.GetVulnLink(cve.GetCve())
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
		cvssV2.Severity = sv
	}

	cvssV2.Score = float32(baseMetricV2.CVSSV2.BaseScore)
	cvssV2.ExploitabilityScore = float32(baseMetricV2.ExploitabilityScore)
	cvssV2.ImpactScore = float32(baseMetricV2.ImpactScore)

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
		cvssV3.Severity = sv
	}

	cvssV3.Score = float32(baseMetricV3.CVSSV3.BaseScore)
	cvssV3.ExploitabilityScore = float32(baseMetricV3.ExploitabilityScore)
	cvssV3.ImpactScore = float32(baseMetricV3.ImpactScore)

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

// ImageCVEToEmbeddedVulnerability coverts a Proto CVEs to Embedded Vuln
// It converts all the fields except except Fixed By which gets set depending on the CVE
func ImageCVEToEmbeddedVulnerability(vuln *storage.ImageCVE) *storage.EmbeddedVulnerability {
	embeddedCVE := &storage.EmbeddedVulnerability{
		Cve:                   vuln.GetCveBaseInfo().GetCve(),
		Cvss:                  vuln.GetCvss(),
		Summary:               vuln.GetCveBaseInfo().GetSummary(),
		Link:                  vuln.GetCveBaseInfo().GetLink(),
		CvssV2:                vuln.GetCveBaseInfo().GetCvssV2(),
		CvssV3:                vuln.GetCveBaseInfo().GetCvssV3(),
		PublishedOn:           vuln.GetCveBaseInfo().GetPublishedOn(),
		LastModified:          vuln.GetCveBaseInfo().GetLastModified(),
		FirstSystemOccurrence: vuln.GetCveBaseInfo().GetCreatedAt(),
		Suppressed:            vuln.GetSnoozed(),
		SuppressActivation:    vuln.GetSnoozeStart(),
		SuppressExpiry:        vuln.GetSnoozeExpiry(),
		Severity:              vuln.GetSeverity(),
	}
	if vuln.GetCveBaseInfo().GetCvssV3() != nil {
		embeddedCVE.ScoreVersion = storage.EmbeddedVulnerability_V3
	} else {
		embeddedCVE.ScoreVersion = storage.EmbeddedVulnerability_V2
	}
	embeddedCVE.VulnerabilityType = storage.EmbeddedVulnerability_IMAGE_VULNERABILITY
	embeddedCVE.VulnerabilityTypes = []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY}
	return embeddedCVE
}

// NodeCVEToNodeVulnerability coverts a Proto CVEs to Embedded node vulnerability.
// It converts all the fields except fields that depend on the node context.
func NodeCVEToNodeVulnerability(protoCVE *storage.NodeCVE) *storage.NodeVulnerability {
	embeddedCVE := &storage.NodeVulnerability{
		CveBaseInfo:  protoCVE.GetCveBaseInfo(),
		Severity:     protoCVE.GetSeverity(),
		Cvss:         protoCVE.GetCvss(),
		Snoozed:      protoCVE.GetSnoozed(),
		SnoozeStart:  protoCVE.GetSnoozeStart(),
		SnoozeExpiry: protoCVE.GetSnoozeExpiry(),
	}
	return embeddedCVE
}

// EmbeddedCVEToProtoCVE converts *storage.EmbeddedVulnerability object to *storage.CVE object
func EmbeddedCVEToProtoCVE(os string, from *storage.EmbeddedVulnerability) *storage.CVE {
	ret := &storage.CVE{
		Type:               embeddedVulnTypeToProtoType(from.GetVulnerabilityType()),
		Id:                 cve.ID(from.GetCve(), os),
		Cvss:               from.GetCvss(),
		Summary:            from.GetSummary(),
		Link:               from.GetLink(),
		PublishedOn:        from.GetPublishedOn(),
		LastModified:       from.GetLastModified(),
		CvssV2:             from.GetCvssV2(),
		CvssV3:             from.GetCvssV3(),
		Suppressed:         from.GetSuppressed(),
		SuppressActivation: from.GetSuppressActivation(),
		SuppressExpiry:     from.GetSuppressExpiry(),
	}
	if ret.CvssV3 != nil {
		ret.ScoreVersion = storage.CVE_V3
		ret.ImpactScore = from.GetCvssV3().GetImpactScore()
	} else if ret.CvssV2 != nil {
		ret.ScoreVersion = storage.CVE_V2
		ret.ImpactScore = from.GetCvssV2().GetImpactScore()
	}

	ret.Severity = from.GetSeverity()
	return ret
}

// EmbeddedVulnerabilityToImageCVE converts *storage.EmbeddedVulnerability object to *storage.ImageCVE object
func EmbeddedVulnerabilityToImageCVE(os string, from *storage.EmbeddedVulnerability) *storage.ImageCVE {
	ret := &storage.ImageCVE{
		Id:              cve.ID(from.GetCve(), os),
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
		},
		Cvss:         from.GetCvss(),
		Severity:     from.GetSeverity(),
		Snoozed:      from.GetSuppressed(),
		SnoozeStart:  from.GetSuppressActivation(),
		SnoozeExpiry: from.GetSuppressExpiry(),
	}
	if ret.GetCveBaseInfo().GetCvssV3() != nil {
		ret.CveBaseInfo.ScoreVersion = storage.CVEInfo_V3
		ret.ImpactScore = from.GetCvssV3().GetImpactScore()
	} else if ret.GetCveBaseInfo().GetCvssV2() != nil {
		ret.CveBaseInfo.ScoreVersion = storage.CVEInfo_V2
		ret.ImpactScore = from.GetCvssV2().GetImpactScore()
	}
	return ret
}

// EmbeddedVulnerabilityToClusterCVE converts *storage.EmbeddedVulnerability object to *storage.ClusterCVE object
func EmbeddedVulnerabilityToClusterCVE(cveType storage.CVE_CVEType, from *storage.EmbeddedVulnerability) *storage.ClusterCVE {
	ret := &storage.ClusterCVE{
		Id: cve.ID(from.GetCve(), cveType.String()),
		CveBaseInfo: &storage.CVEInfo{
			Cve:          from.GetCve(),
			Summary:      from.GetSummary(),
			Link:         from.GetLink(),
			PublishedOn:  from.GetPublishedOn(),
			CreatedAt:    from.GetFirstSystemOccurrence(),
			LastModified: from.GetLastModified(),
			CvssV2:       from.GetCvssV2(),
			CvssV3:       from.GetCvssV3(),
		},
		Cvss:         from.GetCvss(),
		Severity:     from.GetSeverity(),
		Snoozed:      from.GetSuppressed(),
		SnoozeStart:  from.GetSuppressActivation(),
		SnoozeExpiry: from.GetSuppressExpiry(),
		Type:         cveType,
	}
	if ret.GetCveBaseInfo().GetCvssV3() != nil {
		ret.CveBaseInfo.ScoreVersion = storage.CVEInfo_V3
		ret.ImpactScore = from.GetCvssV3().GetImpactScore()
	} else if ret.GetCveBaseInfo().GetCvssV2() != nil {
		ret.CveBaseInfo.ScoreVersion = storage.CVEInfo_V2
		ret.ImpactScore = from.GetCvssV2().GetImpactScore()
	}
	return ret
}

// NodeVulnerabilityToNodeCVE converts *storage.NodeVulnerability object to *storage.NodeCVE object
func NodeVulnerabilityToNodeCVE(os string, from *storage.NodeVulnerability) *storage.NodeCVE {
	ret := &storage.NodeCVE{
		Id:              cve.ID(from.GetCveBaseInfo().GetCve(), os),
		CveBaseInfo:     from.GetCveBaseInfo(),
		Cvss:            from.GetCvss(),
		OperatingSystem: os,
		Severity:        from.GetSeverity(),
		Snoozed:         from.GetSnoozed(),
		SnoozeStart:     from.GetSnoozeStart(),
		SnoozeExpiry:    from.GetSnoozeExpiry(),
	}
	if from.GetCveBaseInfo().GetCvssV3() != nil {
		ret.ImpactScore = from.GetCveBaseInfo().GetCvssV3().GetImpactScore()
	} else if from.GetCveBaseInfo().GetCvssV2() != nil {
		ret.ImpactScore = from.GetCveBaseInfo().GetCvssV2().GetImpactScore()
	}
	return ret
}

// EmbeddedCVEsToProtoCVEs converts *storage.EmbeddedVulnerability to *storage.CVE
func EmbeddedCVEsToProtoCVEs(os string, froms ...*storage.EmbeddedVulnerability) []*storage.CVE {
	ret := make([]*storage.CVE, 0, len(froms))
	for _, from := range froms {
		ret = append(ret, EmbeddedCVEToProtoCVE(os, from))
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

// CVEScoreVersionToEmbeddedScoreVersion converts versions between cve protos.
func CVEScoreVersionToEmbeddedScoreVersion(v storage.CVE_ScoreVersion) storage.EmbeddedVulnerability_ScoreVersion {
	switch v {
	case storage.CVE_V2:
		return storage.EmbeddedVulnerability_V2
	case storage.CVE_V3:
		return storage.EmbeddedVulnerability_V3
	default:
		return storage.EmbeddedVulnerability_V2
	}
}
