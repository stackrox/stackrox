package storagetov2

import (
	"github.com/stackrox/rox/central/convert/helpers"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
)

func VirtualMachine(vm *storage.VirtualMachine) *v2.VirtualMachine {
	if vm == nil {
		return nil
	}

	vm2 := &v2.VirtualMachine{}
	vm2.SetId(vm.GetId())
	vm2.SetNamespace(vm.GetNamespace())
	vm2.SetName(vm.GetName())
	vm2.SetClusterId(vm.GetClusterId())
	vm2.SetClusterName(vm.GetClusterName())
	vm2.SetVsockCid(vm.GetVsockCid())
	vm2.SetState(convertVirtualMachineState(vm.GetState()))
	vm2.SetLastUpdated(vm.GetLastUpdated())
	vm2.SetScan(VirtualMachineScan(vm.GetScan()))
	return vm2
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

func VirtualMachineScan(scan *storage.VirtualMachineScan) *v2.VirtualMachineScan {
	if scan == nil {
		return nil
	}
	vms := &v2.VirtualMachineScan{}
	vms.SetScanTime(scan.GetScanTime())
	vms.SetOperatingSystem(scan.GetOperatingSystem())
	vms.SetNotes(VirtualMachineScanNotes(scan.GetNotes()))
	vms.SetComponents(EmbeddedVirtualMachineScanComponents(scan.GetComponents()))
	return vms
}

func VirtualMachineScanNotes(notes []storage.VirtualMachineScan_Note) []v2.VirtualMachineScan_Note {
	return helpers.ConvertEnumArray(notes, convertVirtualMachineScanNote)
}

func convertVirtualMachineScanNote(note storage.VirtualMachineScan_Note) v2.VirtualMachineScan_Note {
	switch note {
	case storage.VirtualMachineScan_UNSET:
		return v2.VirtualMachineScan_UNSET
	case storage.VirtualMachineScan_OS_UNKNOWN:
		return v2.VirtualMachineScan_OS_UNKNOWN
	case storage.VirtualMachineScan_OS_UNSUPPORTED:
		return v2.VirtualMachineScan_OS_UNSUPPORTED
	default:
		return v2.VirtualMachineScan_UNSET
	}
}
