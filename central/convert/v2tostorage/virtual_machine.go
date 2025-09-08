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
		DataSource:     VirtualMachineDataSource(scan.GetDataSource()),
		Notes:          convertVirtualMachineScanNotes(scan.GetNotes()),
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
		result.SetTopCvss = &storage.EmbeddedVirtualMachineScanComponent_TopCvss{
			TopCvss: component.GetTopCvss(),
		}
	}

	return result
}

func VirtualMachineLicense(license *v2.License) *storage.EmbeddedVirtualMachineScanComponent_License {
	if license == nil {
		return nil
	}

	return &storage.EmbeddedVirtualMachineScanComponent_License{
		Name: license.GetName(),
		Type: license.GetType(),
		Url:  license.GetUrl(),
	}
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
		Cve:               vuln.GetCve(),
		Summary:           vuln.GetSummary(),
		Link:              vuln.GetLink(),
		VulnerabilityType: convertVirtualMachineVulnerabilityType(vuln.GetVulnerabilityType()),
		Severity:          convertSeverity(vuln.GetSeverity()),
		// CvssV2 is not available in v2 API
		CvssV3:       CvssV3(vuln.GetCvssV3()),
		PublishedOn:  vuln.GetPublishedOn(),
		LastModified: vuln.GetLastModified(),
	}

	if vuln.GetFixedBy() != "" {
		result.SetFixedBy = &storage.EmbeddedVirtualMachineVulnerability_FixedBy{
			FixedBy: vuln.GetFixedBy(),
		}
	}

	return result
}

func convertVirtualMachineVulnerabilityType(vt v2.VulnerabilityType) storage.EmbeddedVirtualMachineVulnerability_VulnerabilityType {
	// These enum mappings work as-is at the time of this writing
	// Any new enums will need to be reflected accordingly
	return storage.EmbeddedVirtualMachineVulnerability_VulnerabilityType(vt)
}

func convertVirtualMachineSourceType(source v2.SourceType) storage.EmbeddedVirtualMachineScanComponent_SourceType {
	switch source {
	case v2.SourceType_OS:
		return storage.EmbeddedVirtualMachineScanComponent_OS
	case v2.SourceType_PYTHON:
		return storage.EmbeddedVirtualMachineScanComponent_PYTHON
	case v2.SourceType_JAVA:
		return storage.EmbeddedVirtualMachineScanComponent_JAVA
	case v2.SourceType_RUBY:
		return storage.EmbeddedVirtualMachineScanComponent_RUBY
	case v2.SourceType_NODEJS:
		return storage.EmbeddedVirtualMachineScanComponent_NODEJS
	case v2.SourceType_GO:
		return storage.EmbeddedVirtualMachineScanComponent_GO
	case v2.SourceType_DOTNETCORERUNTIME:
		return storage.EmbeddedVirtualMachineScanComponent_DOTNETCORERUNTIME
	case v2.SourceType_INFRASTRUCTURE:
		return storage.EmbeddedVirtualMachineScanComponent_INFRASTRUCTURE
	default:
		return storage.EmbeddedVirtualMachineScanComponent_OS
	}
}

func VirtualMachineExecutables(executables []*v2.ScanComponent_Executable) []*storage.EmbeddedVirtualMachineScanComponent_Executable {
	if len(executables) == 0 {
		return nil
	}

	var ret []*storage.EmbeddedVirtualMachineScanComponent_Executable
	for _, executable := range executables {
		if executable == nil {
			continue
		}
		ret = append(ret, VirtualMachineExecutable(executable))
	}

	return ret
}

func VirtualMachineExecutable(executable *v2.ScanComponent_Executable) *storage.EmbeddedVirtualMachineScanComponent_Executable {
	if executable == nil {
		return nil
	}

	return &storage.EmbeddedVirtualMachineScanComponent_Executable{
		Path:         executable.GetPath(),
		Dependencies: executable.GetDependencies(),
	}
}

func VirtualMachineDataSource(ds *v2.DataSource) *storage.VirtualMachineScan_DataSource {
	if ds == nil {
		return nil
	}

	return &storage.VirtualMachineScan_DataSource{
		Id:     ds.GetId(),
		Name:   ds.GetName(),
		Mirror: ds.GetMirror(),
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
