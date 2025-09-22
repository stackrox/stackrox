package v2tostorage

import (
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
)

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
		Vulns:        EmbeddedVulnerabilities(component.GetVulns()),
		RiskScore:    component.GetRiskScore(),
		Architecture: component.GetArchitecture(),
	}

	if component.GetTopCvss() != 0 {
		result.SetTopCvss = &storage.EmbeddedImageScanComponent_TopCvss{
			TopCvss: component.GetTopCvss(),
		}
	}

	return result
}
