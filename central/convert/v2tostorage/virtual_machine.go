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
		Scan:        VirtualMachineScan(vm.GetScan()),
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

func VirtualMachineScan(scan *v2.VirtualMachineScan) *storage.VirtualMachineScan {
	if scan == nil {
		return nil
	}

	return &storage.VirtualMachineScan{
		ScannerVersion: scan.GetScannerVersion(),
		ScanTime:       scan.GetScanTime(),
		Components:     VirtualMachineScanComponents(scan.GetComponents()),
	}
}

func VirtualMachineScanComponents(components []*v2.ScanComponent) []*storage.EmbeddedVirtualMachineScanComponent {
	if len(components) == 0 {
		return nil
	}

	var ret []*storage.EmbeddedVirtualMachineScanComponent
	for _, component := range components {
		if component == nil {
			continue
		}
		ret = append(ret, VirtualMachineScanComponent(component))
	}

	return ret
}

func VirtualMachineScanComponent(component *v2.ScanComponent) *storage.EmbeddedVirtualMachineScanComponent {
	if component == nil {
		return nil
	}

	result := &storage.EmbeddedVirtualMachineScanComponent{
		Name:    component.GetName(),
		Version: component.GetVersion(),
		Vulns:   EmbeddedVirtualMachineVulnerabilities(component.GetVulns()),
	}

	return result
}

func EmbeddedVirtualMachineVulnerabilities(vulns []*v2.EmbeddedVulnerability) []*storage.EmbeddedVirtualMachineVulnerability {
	if len(vulns) == 0 {
		return nil
	}

	var ret []*storage.EmbeddedVirtualMachineVulnerability
	for _, vuln := range vulns {
		if vuln == nil {
			continue
		}
		ret = append(ret, EmbeddedVirtualMachineVulnerability(vuln))
	}

	return ret
}

func EmbeddedVirtualMachineVulnerability(vuln *v2.EmbeddedVulnerability) *storage.EmbeddedVirtualMachineVulnerability {
	if vuln == nil {
		return nil
	}

	result := &storage.EmbeddedVirtualMachineVulnerability{
		Cve: vuln.GetCve(),
	}

	return result
}
