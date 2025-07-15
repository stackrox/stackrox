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

func DataSource(ds *v2.DataSource) *storage.DataSource {
	if ds == nil {
		return nil
	}

	return &storage.DataSource{
		Id:     ds.GetId(),
		Name:   ds.GetName(),
		Mirror: ds.GetMirror(),
	}
}

func ScanComponents(components []*v2.ScanComponent) []*storage.EmbeddedImageScanComponent {
	if len(components) == 0 {
		return nil
	}

	var ret []*storage.EmbeddedImageScanComponent
	for _, component := range components {
		if component == nil {
			continue
		}
		ret = append(ret, ScanComponent(component))
	}

	return ret
}

func ScanComponent(component *v2.ScanComponent) *storage.EmbeddedImageScanComponent {
	if component == nil {
		return nil
	}

	result := &storage.EmbeddedImageScanComponent{
		Name:         component.GetName(),
		Version:      component.GetVersion(),
		License:      License(component.GetLicense()),
		Vulns:        EmbeddedVulnerabilities(component.GetVulns()),
		Source:       convertSourceType(component.GetSource()),
		Location:     component.GetLocation(),
		RiskScore:    component.GetRiskScore(),
		FixedBy:      component.GetFixedBy(),
		Executables:  Executables(component.GetExecutables()),
		Architecture: component.GetArchitecture(),
	}

	if component.GetTopCvss() != 0 {
		result.SetTopCvss = &storage.EmbeddedImageScanComponent_TopCvss{
			TopCvss: component.GetTopCvss(),
		}
	}

	return result
}

func License(license *v2.License) *storage.License {
	if license == nil {
		return nil
	}

	return &storage.License{
		Name: license.GetName(),
		Type: license.GetType(),
		Url:  license.GetUrl(),
	}
}

func EmbeddedVulnerabilities(vulns []*v2.EmbeddedVulnerability) []*storage.EmbeddedVulnerability {
	if len(vulns) == 0 {
		return nil
	}

	var ret []*storage.EmbeddedVulnerability
	for _, vuln := range vulns {
		if vuln == nil {
			continue
		}
		ret = append(ret, EmbeddedVulnerability(vuln))
	}

	return ret
}

func EmbeddedVulnerability(vuln *v2.EmbeddedVulnerability) *storage.EmbeddedVulnerability {
	if vuln == nil {
		return nil
	}

	result := &storage.EmbeddedVulnerability{
		Cve:               vuln.GetCve(),
		Summary:           vuln.GetSummary(),
		Link:              vuln.GetLink(),
		VulnerabilityType: convertVulnerabilityType(vuln.GetVulnerabilityType()),
		Severity:          convertSeverity(vuln.GetSeverity()),
		// CvssV2 is not available in v2 API
		CvssV3:       CvssV3(vuln.GetCvssV3()),
		PublishedOn:  vuln.GetPublishedOn(),
		LastModified: vuln.GetLastModified(),
		// ScoreVersion: convertScoreVersion(vuln.GetScoreVersion()),
		// VulnerabilityTypes: convertVulnerabilityTypes(vuln.GetVulnerabilityTypes()),
	}

	if vuln.GetFixedBy() != "" {
		result.SetFixedBy = &storage.EmbeddedVulnerability_FixedBy{
			FixedBy: vuln.GetFixedBy(),
		}
	}

	return result
}

func CvssV2(cvss *v2.CVSSV2) *storage.CVSSV2 {
	if cvss == nil {
		return nil
	}

	return &storage.CVSSV2{
		Vector:              cvss.GetVector(),
		AttackVector:        convertAttackVector(cvss.GetAttackVector()),
		AccessComplexity:    convertAccessComplexity(cvss.GetAccessComplexity()),
		Authentication:      convertAuthentication(cvss.GetAuthentication()),
		Confidentiality:     convertImpact(cvss.GetConfidentiality()),
		Integrity:           convertImpact(cvss.GetIntegrity()),
		Availability:        convertImpact(cvss.GetAvailability()),
		ExploitabilityScore: cvss.GetExploitabilityScore(),
		ImpactScore:         cvss.GetImpactScore(),
		Score:               cvss.GetScore(),
		Severity:            convertCVSSV2Severity(cvss.GetSeverity()),
	}
}

func CvssV3(cvss *v2.CVSSV3) *storage.CVSSV3 {
	if cvss == nil {
		return nil
	}

	return &storage.CVSSV3{
		Vector:              cvss.GetVector(),
		ExploitabilityScore: cvss.GetExploitabilityScore(),
		ImpactScore:         cvss.GetImpactScore(),
		AttackVector:        convertAttackVectorV3(cvss.GetAttackVector()),
		AttackComplexity:    convertComplexity(cvss.GetAttackComplexity()),
		PrivilegesRequired:  convertPrivileges(cvss.GetPrivilegesRequired()),
		UserInteraction:     convertUserInteraction(cvss.GetUserInteraction()),
		Scope:               convertScope(cvss.GetScope()),
		Confidentiality:     convertImpactV3(cvss.GetConfidentiality()),
		Integrity:           convertImpactV3(cvss.GetIntegrity()),
		Availability:        convertImpactV3(cvss.GetAvailability()),
		Score:               cvss.GetScore(),
		Severity:            convertCVSSV3Severity(cvss.GetSeverity()),
	}
}

func Executables(executables []*v2.ScanComponent_Executable) []*storage.EmbeddedImageScanComponent_Executable {
	if len(executables) == 0 {
		return nil
	}

	var ret []*storage.EmbeddedImageScanComponent_Executable
	for _, executable := range executables {
		if executable == nil {
			continue
		}
		ret = append(ret, Executable(executable))
	}

	return ret
}

func Executable(executable *v2.ScanComponent_Executable) *storage.EmbeddedImageScanComponent_Executable {
	if executable == nil {
		return nil
	}

	return &storage.EmbeddedImageScanComponent_Executable{
		Path:         executable.GetPath(),
		Dependencies: executable.GetDependencies(),
	}
}

func convertSourceType(source v2.SourceType) storage.SourceType {
	switch source {
	case v2.SourceType_OS:
		return storage.SourceType_OS
	case v2.SourceType_PYTHON:
		return storage.SourceType_PYTHON
	case v2.SourceType_JAVA:
		return storage.SourceType_JAVA
	case v2.SourceType_RUBY:
		return storage.SourceType_RUBY
	case v2.SourceType_NODEJS:
		return storage.SourceType_NODEJS
	case v2.SourceType_GO:
		return storage.SourceType_GO
	case v2.SourceType_DOTNETCORERUNTIME:
		return storage.SourceType_DOTNETCORERUNTIME
	case v2.SourceType_INFRASTRUCTURE:
		return storage.SourceType_INFRASTRUCTURE
	default:
		return storage.SourceType_OS
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

// Placeholder conversion functions for vulnerability-related enums
// These would need to be implemented based on the actual storage vulnerability types
func convertVulnerabilityType(vt v2.VulnerabilityType) storage.EmbeddedVulnerability_VulnerabilityType {
	// Implementation depends on the actual vulnerability type mappings
	return storage.EmbeddedVulnerability_VulnerabilityType(vt)
}

func convertSeverity(severity v2.VulnerabilitySeverity) storage.VulnerabilitySeverity {
	return storage.VulnerabilitySeverity(severity)
}

func convertCVSSV2Severity(severity v2.CVSSV2_Severity) storage.CVSSV2_Severity {
	return storage.CVSSV2_Severity(severity)
}

func convertCVSSV3Severity(severity v2.CVSSV3_Severity) storage.CVSSV3_Severity {
	return storage.CVSSV3_Severity(severity)
}

func convertAttackVector(av v2.CVSSV2_AttackVector) storage.CVSSV2_AttackVector {
	return storage.CVSSV2_AttackVector(av)
}

func convertAccessComplexity(ac v2.CVSSV2_AccessComplexity) storage.CVSSV2_AccessComplexity {
	return storage.CVSSV2_AccessComplexity(ac)
}

func convertAuthentication(auth v2.CVSSV2_Authentication) storage.CVSSV2_Authentication {
	return storage.CVSSV2_Authentication(auth)
}

func convertImpact(impact v2.CVSSV2_Impact) storage.CVSSV2_Impact {
	return storage.CVSSV2_Impact(impact)
}

func convertAttackVectorV3(av v2.CVSSV3_AttackVector) storage.CVSSV3_AttackVector {
	return storage.CVSSV3_AttackVector(av)
}

func convertComplexity(ac v2.CVSSV3_Complexity) storage.CVSSV3_Complexity {
	return storage.CVSSV3_Complexity(ac)
}

func convertPrivileges(pr v2.CVSSV3_Privileges) storage.CVSSV3_Privileges {
	return storage.CVSSV3_Privileges(pr)
}

func convertUserInteraction(ui v2.CVSSV3_UserInteraction) storage.CVSSV3_UserInteraction {
	return storage.CVSSV3_UserInteraction(ui)
}

func convertScope(scope v2.CVSSV3_Scope) storage.CVSSV3_Scope {
	return storage.CVSSV3_Scope(scope)
}

func convertImpactV3(impact v2.CVSSV3_Impact) storage.CVSSV3_Impact {
	return storage.CVSSV3_Impact(impact)
}
