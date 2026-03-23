package internaltostorage

import (
	virtualMachineV1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgVM "github.com/stackrox/rox/pkg/virtualmachine"
)

// VirtualMachineV2 converts a sensor-internal VirtualMachine to a storage VirtualMachineV2.
func VirtualMachineV2(vm *virtualMachineV1.VirtualMachine) *storage.VirtualMachineV2 {
	if vm == nil {
		return nil
	}

	guestOS := vm.GetFacts()[pkgVM.GuestOSKey]
	if guestOS == "" {
		guestOS = pkgVM.UnknownGuestOS
	}

	return &storage.VirtualMachineV2{
		Id:        vm.GetId(),
		Name:      vm.GetName(),
		Namespace: vm.GetNamespace(),
		ClusterId: vm.GetClusterId(),
		Facts:     vm.GetFacts(),
		GuestOs:   guestOS,
		State:     convertVirtualMachineV2State(vm.GetState()),
		VsockCid:  vm.GetVsockCid(),
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
