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
		DataSource:     VirtualMachineDataSource(scan.GetDataSource()),
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
		Name:         component.GetName(),
		Version:      component.GetVersion(),
		License:      VirtualMachineLicense(component.GetLicense()),
		Vulns:        EmbeddedVirtualMachineVulnerabilities(component.GetVulns()),
		Source:       convertVirtualMachineSourceType(component.GetSource()),
		Location:     component.GetLocation(),
		RiskScore:    component.GetRiskScore(),
		FixedBy:      component.GetFixedBy(),
		Executables:  VirtualMachineExecutables(component.GetExecutables()),
		Architecture: component.GetArchitecture(),
	}

	if component.GetTopCvss() != 0 {
		result.SetTopCvss = &v2.ScanComponent_TopCvss{
			TopCvss: component.GetTopCvss(),
		}
	}

	return result
}

func VirtualMachineLicense(license *storage.EmbeddedVirtualMachineScanComponent_License) *v2.License {
	if license == nil {
		return nil
	}

	return &v2.License{
		Name: license.GetName(),
		Type: license.GetType(),
		Url:  license.GetUrl(),
	}
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
		Cve:               vuln.GetCve(),
		Summary:           vuln.GetSummary(),
		Link:              vuln.GetLink(),
		VulnerabilityType: convertVirtualMachineVulnerabilityType(vuln.GetVulnerabilityType()),
		Severity:          convertSeverity(vuln.GetSeverity()),
		CvssV3:            CvssV3(vuln.GetCvssV3()),
		PublishedOn:       vuln.GetPublishedOn(),
		LastModified:      vuln.GetLastModified(),
	}

	if vuln.GetFixedBy() != "" {
		result.SetFixedBy = &v2.EmbeddedVulnerability_FixedBy{
			FixedBy: vuln.GetFixedBy(),
		}
	}

	return result
}

func convertVirtualMachineVulnerabilityType(vt storage.EmbeddedVirtualMachineVulnerability_VulnerabilityType) v2.VulnerabilityType {
	// These enum mappings work as-is at the time of this writing
	// Any new enums will need to be reflected accordingly
	return v2.VulnerabilityType(vt)
}

func convertVirtualMachineSourceType(source storage.EmbeddedVirtualMachineScanComponent_SourceType) v2.SourceType {
	switch source {
	case storage.EmbeddedVirtualMachineScanComponent_OS:
		return v2.SourceType_OS
	case storage.EmbeddedVirtualMachineScanComponent_PYTHON:
		return v2.SourceType_PYTHON
	case storage.EmbeddedVirtualMachineScanComponent_JAVA:
		return v2.SourceType_JAVA
	case storage.EmbeddedVirtualMachineScanComponent_RUBY:
		return v2.SourceType_RUBY
	case storage.EmbeddedVirtualMachineScanComponent_NODEJS:
		return v2.SourceType_NODEJS
	case storage.EmbeddedVirtualMachineScanComponent_GO:
		return v2.SourceType_GO
	case storage.EmbeddedVirtualMachineScanComponent_DOTNETCORERUNTIME:
		return v2.SourceType_DOTNETCORERUNTIME
	case storage.EmbeddedVirtualMachineScanComponent_INFRASTRUCTURE:
		return v2.SourceType_INFRASTRUCTURE
	default:
		return v2.SourceType_OS
	}
}

func VirtualMachineExecutables(executables []*storage.EmbeddedVirtualMachineScanComponent_Executable) []*v2.ScanComponent_Executable {
	if len(executables) == 0 {
		return nil
	}

	var ret []*v2.ScanComponent_Executable
	for _, executable := range executables {
		if executable == nil {
			continue
		}
		ret = append(ret, VirtualMachineExecutable(executable))
	}

	return ret
}

func VirtualMachineExecutable(executable *storage.EmbeddedVirtualMachineScanComponent_Executable) *v2.ScanComponent_Executable {
	if executable == nil {
		return nil
	}

	return &v2.ScanComponent_Executable{
		Path:         executable.GetPath(),
		Dependencies: executable.GetDependencies(),
	}
}

func VirtualMachineDataSource(ds *storage.VirtualMachineScan_DataSource) *v2.DataSource {
	if ds == nil {
		return nil
	}

	return &v2.DataSource{
		Id:     ds.GetId(),
		Name:   ds.GetName(),
		Mirror: ds.GetMirror(),
	}
}
