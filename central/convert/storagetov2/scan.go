package storagetov2

import (
	"github.com/stackrox/rox/central/convert/helpers"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
)

func EmbeddedVirtualMachineScanComponents(components []*storage.EmbeddedVirtualMachineScanComponent) []*v2.ScanComponent {
	return helpers.ConvertPointerArray(components, EmbeddedVirtualMachineScanComponent)
}

func EmbeddedVirtualMachineScanComponent(cmp *storage.EmbeddedVirtualMachineScanComponent) *v2.ScanComponent {
	if cmp == nil {
		return nil
	}
	result := &v2.ScanComponent{}
	result.SetName(cmp.GetName())
	result.SetVersion(cmp.GetVersion())
	result.SetRiskScore(cmp.GetRiskScore())
	result.SetVulns(VirtualMachineVulnerabilities(cmp.GetVulnerabilities()))
	result.SetSource(convertSourceType(cmp.GetSource()))
	result.SetNotes(scanComponentNotes(cmp.GetNotes()))
	if cmp.GetSetTopCvss() != nil {
		result.Set_TopCvss(cmp.GetTopCvss())
	}
	return result
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

func scanComponentNotes(notes []storage.EmbeddedVirtualMachineScanComponent_Note) []v2.ScanComponent_Note {
	return helpers.ConvertEnumArray(notes, convertScanComponentNoteType)
}

func convertScanComponentNoteType(note storage.EmbeddedVirtualMachineScanComponent_Note) v2.ScanComponent_Note {
	switch note {
	case storage.EmbeddedVirtualMachineScanComponent_UNSCANNED:
		return v2.ScanComponent_UNSCANNED
	default:
		return v2.ScanComponent_UNSPECIFIED
	}
}
