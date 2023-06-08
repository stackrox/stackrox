// This file was originally generated with
// //go:generate cp ../../../central/cve/converter/utils/convert_utils.go

package cvehelper

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cve"
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

		// In dackbox, when reading out the image vulnerabilities, severity is overwritten during merge.
		Severity: protoCVE.GetSeverity(),
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
func EmbeddedCVEToProtoCVE(os string, from *storage.EmbeddedVulnerability, postgresEnabled bool) *storage.CVE {
	ret := &storage.CVE{
		Type:               embeddedVulnTypeToProtoType(from.GetVulnerabilityType()),
		Id:                 from.GetCve(),
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
	if postgresEnabled {
		ret.Id = ID(ret.Id, os)
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
