// This file was originally generated with
// //go:generate cp ../../../central/cve/converter/utils/convert_utils.go

package cvehelper

import (
	"github.com/stackrox/rox/generated/storage"
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
