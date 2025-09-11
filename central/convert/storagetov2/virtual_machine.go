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
		Scan:        VirtualMachineScan(vm.GetScan()),
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

func VirtualMachineScan(scan *storage.VirtualMachineScan) *v2.VirtualMachineScan {
	if scan == nil {
		return nil
	}

	return &v2.VirtualMachineScan{
		ScannerVersion: scan.GetScannerVersion(),
		ScanTime:       scan.GetScanTime(),
		Components:     VirtualMachineScanComponents(scan.GetComponents()),
	}
}

func VirtualMachineScanComponents(components []*storage.EmbeddedVirtualMachineScanComponent) []*v2.ScanComponent {
	if len(components) == 0 {
		return nil
	}

	var ret []*v2.ScanComponent
	for _, component := range components {
		if component == nil {
			continue
		}
		ret = append(ret, VirtualMachineScanComponent(component))
	}

	return ret
}

func VirtualMachineScanComponent(component *storage.EmbeddedVirtualMachineScanComponent) *v2.ScanComponent {
	if component == nil {
		return nil
	}

	result := &v2.ScanComponent{
		Name:    component.GetName(),
		Version: component.GetVersion(),
		Vulns:   EmbeddedVirtualMachineVulnerabilities(component.GetVulns()),
	}

	return result
}

func EmbeddedVirtualMachineVulnerabilities(vulns []*storage.EmbeddedVirtualMachineVulnerability) []*v2.EmbeddedVulnerability {
	if len(vulns) == 0 {
		return nil
	}

	var ret []*v2.EmbeddedVulnerability
	for _, vuln := range vulns {
		if vuln == nil {
			continue
		}
		ret = append(ret, EmbeddedVirtualMachineVulnerability(vuln))
	}

	return ret
}

func EmbeddedVirtualMachineVulnerability(vuln *storage.EmbeddedVirtualMachineVulnerability) *v2.EmbeddedVulnerability {
	if vuln == nil {
		return nil
	}

	result := &v2.EmbeddedVulnerability{
		Cve: vuln.GetCve(),
	}

	return result
}
