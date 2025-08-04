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
		Scan:        VirtualMachineScan(vm.GetScan()),
		LastUpdated: vm.GetLastUpdated(),
	}
}

func VirtualMachineScan(scan *v2.VirtualMachineScan) *storage.VirtualMachineScan {
	if scan == nil {
		return nil
	}

	return &storage.VirtualMachineScan{
		ScannerVersion: scan.GetScannerVersion(),
		ScanTime:       scan.GetScanTime(),
		Components:     ScanComponents(scan.GetComponents()),
		DataSource:     DataSource(scan.GetDataSource()),
		Notes:          convertVirtualMachineScanNotes(scan.GetNotes()),
	}
}

func convertVirtualMachineScanNotes(notes []v2.VirtualMachineScan_Note) []storage.VirtualMachineScan_Note {
	if len(notes) == 0 {
		return nil
	}

	var ret []storage.VirtualMachineScan_Note
	for _, note := range notes {
		ret = append(ret, convertVirtualMachineScanNote(note))
	}

	return ret
}

func convertVirtualMachineScanNote(note v2.VirtualMachineScan_Note) storage.VirtualMachineScan_Note {
	switch note {
	case v2.VirtualMachineScan_UNSET:
		return storage.VirtualMachineScan_UNSET
	case v2.VirtualMachineScan_OS_UNAVAILABLE:
		return storage.VirtualMachineScan_OS_UNAVAILABLE
	case v2.VirtualMachineScan_PARTIAL_SCAN_DATA:
		return storage.VirtualMachineScan_PARTIAL_SCAN_DATA
	case v2.VirtualMachineScan_OS_CVES_UNAVAILABLE:
		return storage.VirtualMachineScan_OS_CVES_UNAVAILABLE
	case v2.VirtualMachineScan_OS_CVES_STALE:
		return storage.VirtualMachineScan_OS_CVES_STALE
	case v2.VirtualMachineScan_LANGUAGE_CVES_UNAVAILABLE:
		return storage.VirtualMachineScan_LANGUAGE_CVES_UNAVAILABLE
	case v2.VirtualMachineScan_CERTIFIED_RHEL_SCAN_UNAVAILABLE:
		return storage.VirtualMachineScan_CERTIFIED_RHEL_SCAN_UNAVAILABLE
	default:
		return storage.VirtualMachineScan_UNSET
	}
}
