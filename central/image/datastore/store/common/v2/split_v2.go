package common

import (
	"github.com/stackrox/rox/central/cve/converter/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scancomponent"
)

func splitComponentsV2(parts ImageParts) []ComponentParts {
	ret := make([]ComponentParts, 0, len(parts.Image.GetScan().GetComponents()))
	for _, component := range parts.Image.GetScan().GetComponents() {
		generatedComponentV2 := GenerateImageComponentV2(parts.Image.GetScan().GetOperatingSystem(), parts.Image, component)

		cp := ComponentParts{
			ComponentV2: generatedComponentV2,
			Children:    splitCVEsV2(parts.Image.Id, generatedComponentV2.GetId(), component),
		}

		ret = append(ret, cp)
	}

	return ret
}

func splitCVEsV2(imageID string, componentID string, embedded *storage.EmbeddedImageScanComponent) []CVEParts {
	ret := make([]CVEParts, 0, len(embedded.GetVulns()))
	for cveIndex, cve := range embedded.GetVulns() {
		convertedCVE := utils.EmbeddedVulnerabilityToImageCVEV2(imageID, componentID, cveIndex, cve)

		cp := CVEParts{
			CVEV2: convertedCVE,
		}
		ret = append(ret, cp)
	}

	return ret
}

// GenerateImageComponentV2 returns top-level image component from embedded component.
func GenerateImageComponentV2(os string, image *storage.Image, from *storage.EmbeddedImageScanComponent) *storage.ImageComponentV2 {
	ret := &storage.ImageComponentV2{
		Id:              scancomponent.ComponentIDV2(from.GetName(), from.GetVersion(), from.GetArchitecture(), image.GetId()),
		Name:            from.GetName(),
		Version:         from.GetVersion(),
		Source:          from.GetSource(),
		FixedBy:         from.GetFixedBy(),
		RiskScore:       from.GetRiskScore(),
		Priority:        from.GetPriority(),
		OperatingSystem: os,
		ImageId:         image.GetId(),
		Location:        from.GetLocation(),
		Architecture:    from.GetArchitecture(),
	}

	if from.GetSetTopCvss() != nil {
		ret.SetTopCvss = &storage.ImageComponentV2_TopCvss{TopCvss: from.GetTopCvss()}
	}

	if from.HasLayerIndex != nil {
		ret.HasLayerIndex = &storage.ImageComponentV2_LayerIndex{
			LayerIndex: from.GetLayerIndex(),
		}
	}

	return ret
}
