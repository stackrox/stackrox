package internaltostorage

import (
	virtualMachineV1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/generated/storage"
)

func VirtualMachine(virtualMachine *virtualMachineV1.VirtualMachine) *storage.VirtualMachine {
	if virtualMachine == nil {
		return nil
	}
	id := virtualMachine.GetId()
	namespace := virtualMachine.GetNamespace()
	name := virtualMachine.GetName()
	clusterId := virtualMachine.GetClusterId()
	vsockCid := virtualMachine.GetVsockCid()
	state := convertVirtualMachineState(virtualMachine.GetState())
	return storage.VirtualMachine_builder{
		Id:        &id,
		Namespace: &namespace,
		Name:      &name,
		ClusterId: &clusterId,
		VsockCid:  &vsockCid,
		State:     &state,
	}.Build()
}

func convertVirtualMachineState(state virtualMachineV1.VirtualMachine_State) storage.VirtualMachine_State {
	switch state {
	case virtualMachineV1.VirtualMachine_UNKNOWN:
		return storage.VirtualMachine_UNKNOWN
	case virtualMachineV1.VirtualMachine_STOPPED:
		return storage.VirtualMachine_STOPPED
	case virtualMachineV1.VirtualMachine_RUNNING:
		return storage.VirtualMachine_RUNNING
	default:
		return storage.VirtualMachine_UNKNOWN
	}
}
