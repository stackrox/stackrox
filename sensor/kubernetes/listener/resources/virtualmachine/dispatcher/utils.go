package dispatcher

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	virtualMachineV1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/virtualmachine"
	sensorVirtualMachine "github.com/stackrox/rox/sensor/common/virtualmachine"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getVirtualMachineOwnerReference(owners []metav1.OwnerReference) (*metav1.OwnerReference, bool) {
	for i := range owners {
		// There should be only one VirtualMachine OwnerReference
		// VirtualMachines and VirtualMachineInstances map 1:1
		if owners[i].Kind == virtualmachine.VirtualMachine.Kind {
			return &owners[i], true
		}
	}
	return nil, false
}

func getVirtualMachineState(vm *sensorVirtualMachine.Info) virtualMachineV1.VirtualMachine_State {
	return sensorVirtualMachine.StateFromInfo(vm)
}

func getVirtualMachineVSockCID(vm *sensorVirtualMachine.Info) (int32, bool) {
	return sensorVirtualMachine.VSockCIDFromInfo(vm)
}

type discoveredFactsStore interface {
	GetDiscoveredFacts(id sensorVirtualMachine.VMID) map[string]string
}

func getFacts(vm *sensorVirtualMachine.Info, discoveredStore discoveredFactsStore) map[string]string {
	var discoveredFacts map[string]string
	if discoveredStore != nil {
		discoveredFacts = discoveredStore.GetDiscoveredFacts(vm.ID)
	}
	return sensorVirtualMachine.BuildFacts(vm, discoveredFacts)
}

func createEvent(action central.ResourceAction, clusterID string, vm *sensorVirtualMachine.Info, discoveredStore discoveredFactsStore) *central.SensorEvent {
	if vm == nil {
		return nil
	}
	vSockCID, vSockCIDSet := getVirtualMachineVSockCID(vm)
	return &central.SensorEvent{
		Id:     string(vm.ID),
		Action: action,
		Resource: &central.SensorEvent_VirtualMachine{
			VirtualMachine: &virtualMachineV1.VirtualMachine{
				Id:          string(vm.ID),
				Namespace:   vm.Namespace,
				Name:        vm.Name,
				ClusterId:   clusterID,
				VsockCid:    vSockCID,
				VsockCidSet: vSockCIDSet,
				State:       getVirtualMachineState(vm),
				Facts:       getFacts(vm, discoveredStore),
			},
		},
	}
}
