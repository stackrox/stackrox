package storagetov2

import (
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
)

func EmbeddedVirtualMachineScanComponents(components []*storage.EmbeddedVirtualMachineScanComponent) []*v2.ScanComponent {
	if components == nil {
		return nil
	}
	result := make([]*v2.ScanComponent, 0, len(components))
	for _, cmp := range components {
		if cmp == nil {
			continue
		}
		result = append(result, EmbeddedVirtualMachineScanComponent(cmp))
	}
	return result
}

func EmbeddedVirtualMachineScanComponent(cmp *storage.EmbeddedVirtualMachineScanComponent) *v2.ScanComponent {
	if cmp == nil {
		return nil
	}
	result := &v2.ScanComponent{
		Name:      cmp.GetName(),
		Version:   cmp.GetVersion(),
		RiskScore: cmp.GetRiskScore(),
		Vulns:     VirtualMachineVulnerabilities(cmp.GetVulnerabilities()),
	}
	if cmp.GetSetTopCvss() != nil {
		result.SetTopCvss = &v2.ScanComponent_TopCvss{
			TopCvss: cmp.GetTopCvss(),
		}
	}
	return result
}
