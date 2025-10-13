package virtualmachine

import (
	"github.com/stackrox/rox/pkg/k8sapi"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubeVirtV1 "kubevirt.io/api/core/v1"
)

// APIResources for VirtualMachine resources.
var (
	groupVersion         = kubeVirtV1.SchemeGroupVersion
	requiredAPIResources []k8sapi.APIResource

	VirtualMachine = registerAPIResource(v1.APIResource{
		Name:    "virtualmachines",
		Kind:    "VirtualMachine",
		Group:   GetGroupVersion().Group,
		Version: GetGroupVersion().Version,
	})
	VirtualMachineInstance = registerAPIResource(v1.APIResource{
		Name:    "virtualmachineinstances",
		Kind:    "VirtualMachineInstance",
		Group:   GetGroupVersion().Group,
		Version: GetGroupVersion().Version,
	})
)

// GetGroupVersion return the group version that uniquely represents the API set of kubeVirt CRs.
func GetGroupVersion() schema.GroupVersion {
	return groupVersion
}

// GetRequiredResources returns the kubeVirt API resources required by ACS.
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
