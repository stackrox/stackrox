package policyreport

import (
	"github.com/stackrox/rox/pkg/k8sapi"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	groupVersion         = schema.GroupVersion{Group: "wgpolicyk8s.io", Version: "v1alpha2"}
	requiredAPIResources []k8sapi.APIResource

	// PolicyReport is the only resource we currently watch. ClusterPolicyReport
	// support will be added in a follow-up when cluster-scoped reports are needed.
	PolicyReport = registerAPIResource(v1.APIResource{
		Name:    "policyreports",
		Kind:    "PolicyReport",
		Group:   groupVersion.Group,
		Version: groupVersion.Version,
	})
)

// GetGroupVersion returns the group version for wgpolicyk8s.io PolicyReport CRDs.
func GetGroupVersion() schema.GroupVersion {
	return groupVersion
}

// GetRequiredResources returns the PolicyReport API resources required by ACS.
func GetRequiredResources() []k8sapi.APIResource {
	return requiredAPIResources
}

func registerAPIResource(resource v1.APIResource) k8sapi.APIResource {
	r := k8sapi.APIResource{
		APIResource: resource,
	}
	requiredAPIResources = append(requiredAPIResources, r)
	return r
}
