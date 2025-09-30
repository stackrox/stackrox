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

	return &v2.VirtualMachine{
		Id:          vm.GetId(),
		Namespace:   vm.GetNamespace(),
		Name:        vm.GetName(),
		ClusterId:   vm.GetClusterId(),
		ClusterName: vm.GetClusterName(),
		VsockCid:    vm.GetVsockCid(),
		State:       convertVirtualMachineState(vm.GetState()),
		LastUpdated: vm.GetLastUpdated(),
		Scan:        VirtualMachineScan(vm.GetScan()),
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

func VirtualMachineScan(scan *storage.VirtualMachineScan) *v2.VirtualMachineScan {
	if scan == nil {
		return nil
	}
	return &v2.VirtualMachineScan{
		ScanTime:        scan.GetScanTime(),
		OperatingSystem: scan.GetOperatingSystem(),
		Notes:           VirtualMachineScanNotes(scan.GetNotes()),
		Components:      EmbeddedVirtualMachineScanComponents(scan.GetComponents()),
	}
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
