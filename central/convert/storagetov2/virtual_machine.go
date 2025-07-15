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
		Scan:        VirtualMachineScan(vm.GetScan()),
		LastUpdated: vm.GetLastUpdated(),
	}
}

func VirtualMachineScan(scan *storage.VirtualMachineScan) *v2.VirtualMachineScan {
	if scan == nil {
		return nil
	}

	return &v2.VirtualMachineScan{
		ScannerVersion: scan.GetScannerVersion(),
		ScanTime:       scan.GetScanTime(),
		Components:     ScanComponents(scan.GetComponents()),
		DataSource:     DataSource(scan.GetDataSource()),
		Notes:          convertVirtualMachineScanNotes(scan.GetNotes()),
	}
}

func convertVirtualMachineScanNotes(notes []storage.VirtualMachineScan_Note) []v2.VirtualMachineScan_Note {
	if len(notes) == 0 {
		return nil
	}

	var ret []v2.VirtualMachineScan_Note
	for _, note := range notes {
		ret = append(ret, convertVirtualMachineScanNote(note))
	}

	return ret
}

func convertVirtualMachineScanNote(note storage.VirtualMachineScan_Note) v2.VirtualMachineScan_Note {
	switch note {
	case storage.VirtualMachineScan_UNSET:
		return v2.VirtualMachineScan_UNSET
	case storage.VirtualMachineScan_OS_UNAVAILABLE:
		return v2.VirtualMachineScan_OS_UNAVAILABLE
	case storage.VirtualMachineScan_PARTIAL_SCAN_DATA:
		return v2.VirtualMachineScan_PARTIAL_SCAN_DATA
	case storage.VirtualMachineScan_OS_CVES_UNAVAILABLE:
		return v2.VirtualMachineScan_OS_CVES_UNAVAILABLE
	case storage.VirtualMachineScan_OS_CVES_STALE:
		return v2.VirtualMachineScan_OS_CVES_STALE
	case storage.VirtualMachineScan_LANGUAGE_CVES_UNAVAILABLE:
		return v2.VirtualMachineScan_LANGUAGE_CVES_UNAVAILABLE
	case storage.VirtualMachineScan_CERTIFIED_RHEL_SCAN_UNAVAILABLE:
		return v2.VirtualMachineScan_CERTIFIED_RHEL_SCAN_UNAVAILABLE
	default:
		return v2.VirtualMachineScan_UNSET
	}
}
