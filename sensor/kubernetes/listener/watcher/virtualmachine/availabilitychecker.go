package virtualmachine

import (
	"github.com/stackrox/rox/pkg/k8sapi"
	"github.com/stackrox/rox/pkg/virtualmachine"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watcher/availability"
)

type checker interface {
	Available(client.Interface) bool
	AppendToCRDWatcher(availability.CrdWatcher) error
	GetResources() []k8sapi.APIResource
}

// NewAvailabilityChecker creates a new AvailabilityChecker
func NewAvailabilityChecker() checker {
	resources := []k8sapi.APIResource{
		virtualmachine.VirtualMachine,
		virtualmachine.VirtualMachineInstance,
	}
	return availability.New(virtualmachine.GetGroupVersion(), resources)
}
