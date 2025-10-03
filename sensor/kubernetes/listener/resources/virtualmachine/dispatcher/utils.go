package dispatcher

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	virtualMachineV1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/virtualmachine"
	sensorVirtualMachine "github.com/stackrox/rox/sensor/common/virtualmachine"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getVirtualMachineOwnerReference(owners []metav1.OwnerReference) (*metav1.OwnerReference, bool) {
	for _, ref := range owners {
		// There should be only one VirtualMachine OwnerReference
		// VirtualMachines and VirtualMachineInstances map 1:1
		if ref.Kind == virtualmachine.VirtualMachine.Kind {
			return &ref, true
		}
	}
	return nil, false
}

func getVirtualMachineState(vm *sensorVirtualMachine.Info) virtualMachineV1.VirtualMachine_State {
	if vm == nil {
		return virtualMachineV1.VirtualMachine_UNKNOWN
	}
	if vm.Running {
		return virtualMachineV1.VirtualMachine_RUNNING
	}
	return virtualMachineV1.VirtualMachine_STOPPED
}

func getVirtualMachineVSockCID(vm *sensorVirtualMachine.Info) (int32, bool) {
	if vm == nil {
		return int32(0), false
	}
	if vm.VSOCKCID == nil {
		return int32(0), false
	}
	return int32(*vm.VSOCKCID), true
}

func createEvent(action central.ResourceAction, clusterID string, vm *sensorVirtualMachine.Info) *central.SensorEvent {
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
			},
		},
	}
}
