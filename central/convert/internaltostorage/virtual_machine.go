package internaltostorage

import (
	virtualMachineV1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/generated/storage"
)

func VirtualMachine(virtualMachine *virtualMachineV1.VirtualMachine) *storage.VirtualMachine {
	if virtualMachine == nil {
		return nil
	}
	vm := &storage.VirtualMachine{}
	vm.SetId(virtualMachine.GetId())
	vm.SetNamespace(virtualMachine.GetNamespace())
	vm.SetName(virtualMachine.GetName())
	vm.SetClusterId(virtualMachine.GetClusterId())
	vm.SetVsockCid(virtualMachine.GetVsockCid())
	vm.SetState(convertVirtualMachineState(virtualMachine.GetState()))
	return vm
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
