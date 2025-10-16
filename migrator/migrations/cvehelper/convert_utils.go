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
// It converts all the fields except Fixed By which gets set depending on the CVE
func ProtoCVEToEmbeddedCVE(protoCVE *storage.CVE) *storage.EmbeddedVulnerability {
	embeddedCVE := &storage.EmbeddedVulnerability{}
	embeddedCVE.SetCve(protoCVE.GetId())
	embeddedCVE.SetCvss(protoCVE.GetCvss())
	embeddedCVE.SetSummary(protoCVE.GetSummary())
	embeddedCVE.SetLink(protoCVE.GetLink())
	embeddedCVE.SetCvssV2(protoCVE.GetCvssV2())
	embeddedCVE.SetCvssV3(protoCVE.GetCvssV3())
	embeddedCVE.SetPublishedOn(protoCVE.GetPublishedOn())
	embeddedCVE.SetLastModified(protoCVE.GetLastModified())
	embeddedCVE.SetFirstSystemOccurrence(protoCVE.GetCreatedAt())
	embeddedCVE.SetSuppressed(protoCVE.GetSuppressed())
	embeddedCVE.SetSuppressActivation(protoCVE.GetSuppressActivation())
	embeddedCVE.SetSuppressExpiry(protoCVE.GetSuppressExpiry())
	embeddedCVE.SetSeverity(protoCVE.GetSeverity())
	if protoCVE.GetCvssV3() != nil {
		embeddedCVE.SetScoreVersion(storage.EmbeddedVulnerability_V3)
	} else {
		embeddedCVE.SetScoreVersion(storage.EmbeddedVulnerability_V2)
	}
	embeddedCVE.SetVulnerabilityType(protoToEmbeddedVulnType(protoCVE.GetType()))
	for _, vulnType := range protoCVE.GetTypes() {
		embeddedCVE.SetVulnerabilityTypes(append(embeddedCVE.GetVulnerabilityTypes(), protoToEmbeddedVulnType(vulnType)))
	}
	return embeddedCVE
}

// ImageCVEToEmbeddedVulnerability coverts a Proto CVEs to Embedded Vuln
// It converts all the fields except except Fixed By which gets set depending on the CVE
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
	ret := &storage.CVE{}
	ret.SetType(embeddedVulnTypeToProtoType(from.GetVulnerabilityType()))
	ret.SetId(from.GetCve())
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
	if postgresEnabled {
		ret.SetId(ID(ret.GetId(), os))
	}
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

// EmbeddedVulnerabilityToImageCVE converts *storage.EmbeddedVulnerability object to *storage.ImageCVE object
func EmbeddedVulnerabilityToImageCVE(os string, from *storage.EmbeddedVulnerability) *storage.ImageCVE {
	cVEInfo := &storage.CVEInfo{}
	cVEInfo.SetCve(from.GetCve())
	cVEInfo.SetSummary(from.GetSummary())
	cVEInfo.SetLink(from.GetLink())
	cVEInfo.SetPublishedOn(from.GetPublishedOn())
	cVEInfo.SetCreatedAt(from.GetFirstSystemOccurrence())
	cVEInfo.SetLastModified(from.GetLastModified())
	cVEInfo.SetCvssV2(from.GetCvssV2())
	cVEInfo.SetCvssV3(from.GetCvssV3())
	ret := &storage.ImageCVE{}
	ret.SetId(cve.ID(from.GetCve(), os))
	ret.SetOperatingSystem(os)
	ret.SetCveBaseInfo(cVEInfo)
	ret.SetCvss(from.GetCvss())
	ret.SetSeverity(from.GetSeverity())
	ret.SetSnoozed(from.GetSuppressed())
	ret.SetSnoozeStart(from.GetSuppressActivation())
	ret.SetSnoozeExpiry(from.GetSuppressExpiry())
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
