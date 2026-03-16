package internaltostorage

import (
	virtualMachineV1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/generated/storage"
)

// VirtualMachineV2 converts a sensor-internal VirtualMachine to a storage VirtualMachineV2.
func VirtualMachineV2(virtualMachine *virtualMachineV1.VirtualMachine) *storage.VirtualMachineV2 {
	if virtualMachine == nil {
		return nil
	}
	return &storage.VirtualMachineV2{
		Id:        virtualMachine.GetId(),
		Name:      virtualMachine.GetName(),
		Namespace: virtualMachine.GetNamespace(),
		ClusterId: virtualMachine.GetClusterId(),
		Facts:     virtualMachine.GetFacts(),
		GuestOs:   virtualMachine.GetFacts()["guestOS"],
		VsockCid:  virtualMachine.GetVsockCid(),
		State:     convertVirtualMachineV2State(virtualMachine.GetState()),
	}
}

func convertVirtualMachineV2State(state virtualMachineV1.VirtualMachine_State) storage.VirtualMachineV2_State {
	switch state {
	case virtualMachineV1.VirtualMachine_UNKNOWN:
		return storage.VirtualMachineV2_UNKNOWN
	case virtualMachineV1.VirtualMachine_STOPPED:
		return storage.VirtualMachineV2_STOPPED
	case virtualMachineV1.VirtualMachine_RUNNING:
		return storage.VirtualMachineV2_RUNNING
	default:
		return storage.VirtualMachineV2_UNKNOWN
	}
}
