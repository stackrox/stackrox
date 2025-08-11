package v2tostorage

import (
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
)

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
