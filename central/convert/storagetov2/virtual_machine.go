package storagetov2

import (
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
)

func VirtualMachine(vm *storage.VirtualMachine) *v2.VirtualMachine {
	if vm == nil {
		return nil
	}

	return &v2.VirtualMachine{
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

func convertVirtualMachineState(state storage.VirtualMachine_State) v2.VirtualMachine_State {
	switch state {
	case storage.VirtualMachine_UNKNOWN:
		return v2.VirtualMachine_UNKNOWN
	case storage.VirtualMachine_STOPPED:
		return v2.VirtualMachine_STOPPED
	case storage.VirtualMachine_RUNNING:
		return v2.VirtualMachine_RUNNING
	default:
		return v2.VirtualMachine_UNKNOWN
	}
}
