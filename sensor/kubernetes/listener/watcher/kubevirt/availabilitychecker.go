package kubevirt

import (
	"github.com/stackrox/rox/pkg/k8sapi"
	"github.com/stackrox/rox/pkg/kubevirt"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watcher/availability"
)

type checker interface {
	Available(client.Interface) bool
	AppendToCRDWatcher(availability.CrdWatcher) error
	GetResources() []k8sapi.APIResource
}

// NewKubeVirtAvailabilityChecker creates a new AvailabilityChecker
func NewKubeVirtAvailabilityChecker() checker {
	resources := []k8sapi.APIResource{
		kubevirt.VirtualMachine,
	}
	return availability.New(kubevirt.GetGroupVersion(), resources)
}
