package dispatcher

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	pkgVM "github.com/stackrox/rox/pkg/virtualmachine"
	sensorVirtualMachine "github.com/stackrox/rox/sensor/common/virtualmachine"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getVirtualMachineOwnerReference(owners []metav1.OwnerReference) (*metav1.OwnerReference, bool) {
	for _, ref := range owners {
		// There should be only one VirtualMachine OwnerReference
		// VirtualMachines and VirtualMachineInstances map 1:1
		if ref.Kind == pkgVM.VirtualMachine.Kind {
			return &ref, true
		}
	}
	return nil, false
}

func createEvent(action central.ResourceAction, clusterID string, vm *sensorVirtualMachine.Info) *central.SensorEvent {
	return sensorVirtualMachine.SensorEvent(action, clusterID, vm)
}

// attachStoredAgentFacts copies scrape-owned facts onto vm. Informer UPDATEs are
// built from the VMI and would otherwise replace Central's facts map without them.
func attachStoredAgentFacts(store virtualMachineStore, vm *sensorVirtualMachine.Info) {
	if vm == nil {
		return
	}
	stored := store.Get(vm.ID)
	if stored == nil || len(stored.AgentFacts) == 0 {
		return
	}
	vm.AgentFacts = stored.AgentFacts
}
