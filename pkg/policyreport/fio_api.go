package policyreport

import (
	"github.com/stackrox/rox/pkg/k8sapi"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	fioGroupVersion         = schema.GroupVersion{Group: "fileintegrity.openshift.io", Version: "v1alpha1"}
	fioRequiredAPIResources []k8sapi.APIResource

	// FileIntegrityNodeStatus is the FIO resource that reports per-node scan results.
	FileIntegrityNodeStatus = registerFIOAPIResource(v1.APIResource{
		Name:    "fileintegritynodestatuses",
		Kind:    "FileIntegrityNodeStatus",
		Group:   fioGroupVersion.Group,
		Version: fioGroupVersion.Version,
	})
)

// GetFIOGroupVersion returns the group version for FileIntegrity CRDs.
func GetFIOGroupVersion() schema.GroupVersion {
	return fioGroupVersion
}

// GetFIORequiredResources returns the FIO API resources required by ACS.
func GetFIORequiredResources() []k8sapi.APIResource {
	return fioRequiredAPIResources
}

func registerFIOAPIResource(resource v1.APIResource) k8sapi.APIResource {
	r := k8sapi.APIResource{
		APIResource: resource,
	}
	fioRequiredAPIResources = append(fioRequiredAPIResources, r)
	return r
}
