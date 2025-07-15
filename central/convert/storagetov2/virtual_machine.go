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

func DataSource(ds *storage.DataSource) *v2.DataSource {
	if ds == nil {
		return nil
	}

	return &v2.DataSource{
		Id:     ds.GetId(),
		Name:   ds.GetName(),
		Mirror: ds.GetMirror(),
	}
}

func ScanComponents(components []*storage.EmbeddedImageScanComponent) []*v2.ScanComponent {
	if len(components) == 0 {
		return nil
	}

	var ret []*v2.ScanComponent
	for _, component := range components {
		if component == nil {
			continue
		}
		ret = append(ret, ScanComponent(component))
	}

	return ret
}

func ScanComponent(component *storage.EmbeddedImageScanComponent) *v2.ScanComponent {
	if component == nil {
		return nil
	}

	result := &v2.ScanComponent{
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
		result.SetTopCvss = &v2.ScanComponent_TopCvss{
			TopCvss: component.GetTopCvss(),
		}
	}

	return result
}

func License(license *storage.License) *v2.License {
	if license == nil {
		return nil
	}

	return &v2.License{
		Name: license.GetName(),
		Type: license.GetType(),
		Url:  license.GetUrl(),
	}
}

func EmbeddedVulnerabilities(vulns []*storage.EmbeddedVulnerability) []*v2.EmbeddedVulnerability {
	if len(vulns) == 0 {
		return nil
	}

	var ret []*v2.EmbeddedVulnerability
	for _, vuln := range vulns {
		if vuln == nil {
			continue
		}
		ret = append(ret, EmbeddedVulnerability(vuln))
	}

	return ret
}

func EmbeddedVulnerability(vuln *storage.EmbeddedVulnerability) *v2.EmbeddedVulnerability {
	if vuln == nil {
		return nil
	}

	result := &v2.EmbeddedVulnerability{
		Cve:               vuln.GetCve(),
		Summary:           vuln.GetSummary(),
		Link:              vuln.GetLink(),
		VulnerabilityType: convertVulnerabilityType(vuln.GetVulnerabilityType()),
		Severity:          convertSeverity(vuln.GetSeverity()),
		CvssV3:            CvssV3(vuln.GetCvssV3()),
		PublishedOn:       vuln.GetPublishedOn(),
		LastModified:      vuln.GetLastModified(),
		// ScoreVersion: convertScoreVersion(vuln.GetScoreVersion()),
		// VulnerabilityTypes: convertVulnerabilityTypes(vuln.GetVulnerabilityTypes()),
	}

	if vuln.GetFixedBy() != "" {
		result.SetFixedBy = &v2.EmbeddedVulnerability_FixedBy{
			FixedBy: vuln.GetFixedBy(),
		}
	}

	return result
}

func CvssV2(cvss *storage.CVSSV2) *v2.CVSSV2 {
	if cvss == nil {
		return nil
	}

	return &v2.CVSSV2{
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

func CvssV3(cvss *storage.CVSSV3) *v2.CVSSV3 {
	if cvss == nil {
		return nil
	}

	return &v2.CVSSV3{
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

func Executables(executables []*storage.EmbeddedImageScanComponent_Executable) []*v2.ScanComponent_Executable {
	if len(executables) == 0 {
		return nil
	}

	var ret []*v2.ScanComponent_Executable
	for _, executable := range executables {
		if executable == nil {
			continue
		}
		ret = append(ret, Executable(executable))
	}

	return ret
}

func Executable(executable *storage.EmbeddedImageScanComponent_Executable) *v2.ScanComponent_Executable {
	if executable == nil {
		return nil
	}

	return &v2.ScanComponent_Executable{
		Path:         executable.GetPath(),
		Dependencies: executable.GetDependencies(),
	}
}

func convertSourceType(source storage.SourceType) v2.SourceType {
	switch source {
	case storage.SourceType_OS:
		return v2.SourceType_OS
	case storage.SourceType_PYTHON:
		return v2.SourceType_PYTHON
	case storage.SourceType_JAVA:
		return v2.SourceType_JAVA
	case storage.SourceType_RUBY:
		return v2.SourceType_RUBY
	case storage.SourceType_NODEJS:
		return v2.SourceType_NODEJS
	case storage.SourceType_GO:
		return v2.SourceType_GO
	case storage.SourceType_DOTNETCORERUNTIME:
		return v2.SourceType_DOTNETCORERUNTIME
	case storage.SourceType_INFRASTRUCTURE:
		return v2.SourceType_INFRASTRUCTURE
	default:
		return v2.SourceType_OS
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

// Placeholder conversion functions for vulnerability-related enums
// These would need to be implemented based on the actual storage vulnerability types
func convertVulnerabilityType(vt storage.EmbeddedVulnerability_VulnerabilityType) v2.VulnerabilityType {
	// Implementation depends on the actual vulnerability type mappings
	return v2.VulnerabilityType(vt)
}

func convertSeverity(severity storage.VulnerabilitySeverity) v2.VulnerabilitySeverity {
	return v2.VulnerabilitySeverity(severity)
}

func convertCVSSV2Severity(severity storage.CVSSV2_Severity) v2.CVSSV2_Severity {
	return v2.CVSSV2_Severity(severity)
}

func convertCVSSV3Severity(severity storage.CVSSV3_Severity) v2.CVSSV3_Severity {
	return v2.CVSSV3_Severity(severity)
}

func convertAttackVector(av storage.CVSSV2_AttackVector) v2.CVSSV2_AttackVector {
	return v2.CVSSV2_AttackVector(av)
}

func convertAccessComplexity(ac storage.CVSSV2_AccessComplexity) v2.CVSSV2_AccessComplexity {
	return v2.CVSSV2_AccessComplexity(ac)
}

func convertAuthentication(auth storage.CVSSV2_Authentication) v2.CVSSV2_Authentication {
	return v2.CVSSV2_Authentication(auth)
}

func convertImpact(impact storage.CVSSV2_Impact) v2.CVSSV2_Impact {
	return v2.CVSSV2_Impact(impact)
}

func convertAttackVectorV3(av storage.CVSSV3_AttackVector) v2.CVSSV3_AttackVector {
	return v2.CVSSV3_AttackVector(av)
}

func convertComplexity(ac storage.CVSSV3_Complexity) v2.CVSSV3_Complexity {
	return v2.CVSSV3_Complexity(ac)
}

func convertPrivileges(pr storage.CVSSV3_Privileges) v2.CVSSV3_Privileges {
	return v2.CVSSV3_Privileges(pr)
}

func convertUserInteraction(ui storage.CVSSV3_UserInteraction) v2.CVSSV3_UserInteraction {
	return v2.CVSSV3_UserInteraction(ui)
}

func convertScope(scope storage.CVSSV3_Scope) v2.CVSSV3_Scope {
	return v2.CVSSV3_Scope(scope)
}

func convertImpactV3(impact storage.CVSSV3_Impact) v2.CVSSV3_Impact {
	return v2.CVSSV3_Impact(impact)
}
