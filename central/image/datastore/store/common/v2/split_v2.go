package common

import (
	"github.com/stackrox/rox/central/cve/converter/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scancomponent"
)

func splitComponentsV2(parts ImageParts) ([]ComponentParts, error) {
	ret := make([]ComponentParts, 0, len(parts.Image.GetScan().GetComponents()))
	for _, component := range parts.Image.GetScan().GetComponents() {
		generatedComponentV2, err := GenerateImageComponentV2(parts.Image.GetScan().GetOperatingSystem(), parts.Image, component)
		if err != nil {
			return nil, err
		}

		cves, err := splitCVEsV2(parts.Image.GetId(), generatedComponentV2.GetId(), component)
		if err != nil {
			return nil, err
		}

		cp := ComponentParts{
			ComponentV2: generatedComponentV2,
			Children:    cves,
		}

		ret = append(ret, cp)
	}

	return ret, nil
}

func splitCVEsV2(imageID string, componentID string, embedded *storage.EmbeddedImageScanComponent) ([]CVEParts, error) {
	ret := make([]CVEParts, 0, len(embedded.GetVulns()))
	for _, cve := range embedded.GetVulns() {
		convertedCVE, err := utils.EmbeddedVulnerabilityToImageCVEV2(imageID, componentID, cve)
		if err != nil {
			return nil, err
		}

		cp := CVEParts{
			CVEV2: convertedCVE,
		}
		ret = append(ret, cp)
	}

	return ret, nil
}

// GenerateImageComponentV2 returns top-level image component from embedded component.
func GenerateImageComponentV2(os string, image *storage.Image, from *storage.EmbeddedImageScanComponent) (*storage.ImageComponentV2, error) {
	componentID, err := scancomponent.ComponentIDV2(from, image.GetId())
	if err != nil {
		return nil, err
	}

	ret := &storage.ImageComponentV2{
		Id:              componentID,
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

	return ret, nil
}
