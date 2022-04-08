package converter

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

// NvdCVEToProtoCVE converts a nvd.CVEEntry object to *storage.CVE object
// Deprecated: Consider using scanner to fetch CVEs.
func NvdCVEToProtoCVE(nvdCVE *schema.NVDCVEFeedJSON10DefCVEItem, ct CVEType) (*storage.CVE, error) {
	protoCVE := &storage.CVE{
		Id: nvdCVE.CVE.CVEDataMeta.ID,
	}

	protoCVE.Type = ct.ToStorageCVEType()
	if protoCVE.Type == storage.CVE_UNKNOWN_CVE {
		return nil, errors.Errorf("unknown CVE type: %s", ct)
	}

	protoCVE.ScoreVersion = storage.CVE_UNKNOWN
	if nvdCVE.Impact != nil {
		cvssv2, err := nvdCvssv2ToProtoCvssv2(nvdCVE.Impact.BaseMetricV2)
		if err != nil {
			return nil, err
		}
		protoCVE.CvssV2 = cvssv2
		protoCVE.Cvss = cvssv2.Score
		protoCVE.ScoreVersion = storage.CVE_V2

		cvssv3, err := nvdCvssv3ToProtoCvssv3(nvdCVE.Impact.BaseMetricV3)
		if err != nil {
			return nil, err
		}
		protoCVE.CvssV3 = cvssv3
		protoCVE.Cvss = cvssv3.Score
		protoCVE.ScoreVersion = storage.CVE_V3
	}

	if nvdCVE.PublishedDate != "" {
		if ts, err := time.Parse(timeFormat, nvdCVE.PublishedDate); err == nil {
			protoCVE.PublishedOn = protoconv.ConvertTimeToTimestamp(ts)
		}
	}

	if nvdCVE.LastModifiedDate != "" {
		if ts, err := time.Parse(timeFormat, nvdCVE.LastModifiedDate); err == nil {
			protoCVE.LastModified = protoconv.ConvertTimeToTimestamp(ts)
		}
	}

	if nvdCVE.CVE.Description != nil && len(nvdCVE.CVE.Description.DescriptionData) > 0 {
		protoCVE.Summary = nvdCVE.CVE.Description.DescriptionData[0].Value
	}

	protoCVE.Link = scans.GetVulnLink(protoCVE.Id)

	return protoCVE, nil
}

// NvdCVEToEmbeddedCVE converts a nvd.CVEEntry object to *storage.EmbeddedVulnerability object
func NvdCVEToEmbeddedCVE(nvdCVE *schema.NVDCVEFeedJSON10DefCVEItem, ct CVEType) (*storage.EmbeddedVulnerability, error) {
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

	if nvdCVE.Impact != nil {
		cvssv2, err := nvdCvssv2ToProtoCvssv2(nvdCVE.Impact.BaseMetricV2)
		if err != nil {
			return nil, err
		}
		cve.CvssV2 = cvssv2
		cve.Cvss = cvssv2.Score
		cve.ScoreVersion = storage.EmbeddedVulnerability_V2

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

	if nvdCVE.CVE.Description != nil && len(nvdCVE.CVE.Description.DescriptionData) > 0 {
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

// NvdCVEsToProtoCVEs converts NVD CVEs to *storage.CVE objects
func NvdCVEsToProtoCVEs(cves []*schema.NVDCVEFeedJSON10DefCVEItem, ct CVEType) ([]*storage.CVE, error) {
	ret := make([]*storage.CVE, 0, len(cves))
	for _, cve := range cves {
		ev, err := NvdCVEToProtoCVE(cve, ct)
		if err != nil {
			return nil, err
		}
		ret = append(ret, ev)
	}
	return ret, nil
}

// NvdCVEsToEmbeddedCVEs converts  NVD CVEs to *storage.CVE objects
func NvdCVEsToEmbeddedCVEs(cves []*schema.NVDCVEFeedJSON10DefCVEItem, ct CVEType) ([]*storage.EmbeddedVulnerability, error) {
	ret := make([]*storage.EmbeddedVulnerability, 0, len(cves))
	for _, cve := range cves {
		ev, err := NvdCVEToEmbeddedCVE(cve, ct)
		if err != nil {
			return nil, err
		}
		ret = append(ret, ev)
	}
	return ret, nil
}

// ProtoCVEsToEmbeddedCVEs coverts Proto CVEs to Embedded Vulns
func ProtoCVEsToEmbeddedCVEs(protoCVEs []*storage.CVE) ([]*storage.EmbeddedVulnerability, error) {
	embeddedVulns := make([]*storage.EmbeddedVulnerability, 0, len(protoCVEs))
	for _, protoCVE := range protoCVEs {
		embeddedVulns = append(embeddedVulns, ProtoCVEToEmbeddedCVE(protoCVE))
	}
	return embeddedVulns, nil
}

// ProtoCVEToEmbeddedCVE coverts a Proto CVEs to Embedded Vuln
// It converts all the fields except except Fixed By which gets set depending on the CVE
func ProtoCVEToEmbeddedCVE(protoCVE *storage.CVE) *storage.EmbeddedVulnerability {
	embeddedCVE := &storage.EmbeddedVulnerability{
		Cve:                   protoCVE.GetId(),
		Cvss:                  protoCVE.GetCvss(),
		Summary:               protoCVE.GetSummary(),
		Link:                  protoCVE.GetLink(),
		CvssV2:                protoCVE.GetCvssV2(),
		CvssV3:                protoCVE.GetCvssV3(),
		PublishedOn:           protoCVE.GetPublishedOn(),
		LastModified:          protoCVE.GetLastModified(),
		FirstSystemOccurrence: protoCVE.GetCreatedAt(),
		Suppressed:            protoCVE.GetSuppressed(),
		SuppressActivation:    protoCVE.GetSuppressActivation(),
		SuppressExpiry:        protoCVE.GetSuppressExpiry(),
	}
	if protoCVE.CvssV3 != nil {
		embeddedCVE.ScoreVersion = storage.EmbeddedVulnerability_V3
	} else {
		embeddedCVE.ScoreVersion = storage.EmbeddedVulnerability_V2
	}
	embeddedCVE.VulnerabilityType = protoToEmbeddedVulnType(protoCVE.GetType())
	for _, vulnType := range protoCVE.GetTypes() {
		embeddedCVE.VulnerabilityTypes = append(embeddedCVE.VulnerabilityTypes, protoToEmbeddedVulnType(vulnType))
	}
	return embeddedCVE
}

func protoToEmbeddedVulnType(protoCVEType storage.CVE_CVEType) storage.EmbeddedVulnerability_VulnerabilityType {
	switch protoCVEType {
	case storage.CVE_IMAGE_CVE:
		return storage.EmbeddedVulnerability_IMAGE_VULNERABILITY
	case storage.CVE_K8S_CVE:
		return storage.EmbeddedVulnerability_K8S_VULNERABILITY
	case storage.CVE_ISTIO_CVE:
		return storage.EmbeddedVulnerability_ISTIO_VULNERABILITY
	case storage.CVE_NODE_CVE:
		return storage.EmbeddedVulnerability_NODE_VULNERABILITY
	case storage.CVE_OPENSHIFT_CVE:
		return storage.EmbeddedVulnerability_OPENSHIFT_VULNERABILITY
	default:
		return storage.EmbeddedVulnerability_UNKNOWN_VULNERABILITY
	}
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
		OperatingSystem:    os,
	}
	if ret.CvssV3 != nil {
		ret.ScoreVersion = storage.CVE_V3
		ret.ImpactScore = from.GetCvssV3().GetImpactScore()
	} else if ret.CvssV2 != nil {
		ret.ScoreVersion = storage.CVE_V2
		ret.ImpactScore = from.GetCvssV2().GetImpactScore()
	}
	// If the OS is empty, then the OS is unknown and we don't need to save
	// any distro specific settings
	if os == "" {
		return ret
	}
	ret.DistroSpecifics = map[string]*storage.CVE_DistroSpecific{
		os: {
			Severity:     from.GetSeverity(),
			Cvss:         ret.Cvss,
			CvssV2:       ret.CvssV2,
			CvssV3:       ret.CvssV3,
			ScoreVersion: ret.ScoreVersion,
		},
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
	if nvdCVE.Configurations == nil {
		return versions
	}

	for _, node := range nvdCVE.Configurations.Nodes {
		for _, cpeMatch := range node.CPEMatch {
			if cpeMatch.VersionEndExcluding != "" {
				versions = append(versions, cpeMatch.VersionEndExcluding)
			}
		}
	}
	return versions
}
