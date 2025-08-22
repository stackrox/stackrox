package storagetov2

import (
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
)

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
