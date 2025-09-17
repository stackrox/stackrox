package v2tostorage

import (
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
)

func VirtualMachine(vm *v2.VirtualMachine) *storage.VirtualMachine {
	if vm == nil {
		return nil
	}

	return &storage.VirtualMachine{
		Id:          vm.GetId(),
		Namespace:   vm.GetNamespace(),
		Name:        vm.GetName(),
		ClusterId:   vm.GetClusterId(),
		ClusterName: vm.GetClusterName(),
		VsockCid:    vm.GetVsockCid(),
		State:       convertVirtualMachineState(vm.GetState()),
		LastUpdated: vm.GetLastUpdated(),
	}
}

func convertVirtualMachineState(state v2.VirtualMachine_State) storage.VirtualMachine_State {
	switch state {
	case v2.VirtualMachine_UNKNOWN:
		return storage.VirtualMachine_UNKNOWN
	case v2.VirtualMachine_STOPPED:
		return storage.VirtualMachine_STOPPED
	case v2.VirtualMachine_RUNNING:
		return storage.VirtualMachine_RUNNING
	default:
		return storage.VirtualMachine_UNKNOWN
	}
}
