package common

import (
	"github.com/stackrox/rox/central/cve/converter/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/scancomponent"
)

// Split splits the input image into a set of parts.
func Split(image *storage.ImageV2, withComponents bool) (ImagePartsV2, error) {
	if !features.FlattenImageData.Enabled() {
		return ImagePartsV2{}, nil
	}

	parts := ImagePartsV2{
		Image: image.CloneVT(),
	}

	if withComponents {
		var err error
		parts.Children, err = splitComponents(parts)
		if err != nil {
			return ImagePartsV2{}, err
		}
	}

	// Clear components in the top level image.
	if parts.Image.GetScan() != nil {
		parts.Image.Scan.Components = nil
	}
	return parts, nil
}

func splitComponents(parts ImagePartsV2) ([]ComponentPartsV2, error) {
	ret := make([]ComponentPartsV2, 0, len(parts.Image.GetScan().GetComponents()))
	for index, component := range parts.Image.GetScan().GetComponents() {
		generatedComponentV2, err := GenerateImageComponentV2(parts.Image.GetScan().GetOperatingSystem(), parts.Image, index, component)
		if err != nil {
			return nil, err
		}

		cves, err := splitCVEs(parts.Image.GetId(), generatedComponentV2.GetId(), component)
		if err != nil {
			return nil, err
		}

		cp := ComponentPartsV2{
			ComponentV2: generatedComponentV2,
			Children:    cves,
		}

		ret = append(ret, cp)
	}

	return ret, nil
}

func splitCVEs(imageID string, componentID string, embedded *storage.EmbeddedImageScanComponent) ([]CVEPartsV2, error) {
	ret := make([]CVEPartsV2, 0, len(embedded.GetVulns()))
	for index, cve := range embedded.GetVulns() {
		convertedCVE, err := utils.EmbeddedVulnerabilityToImageCVEV2(imageID, componentID, index, cve)
		if err != nil {
			return nil, err
		}

		cp := CVEPartsV2{
			CVEV2: convertedCVE,
		}
		ret = append(ret, cp)
	}

	return ret, nil
}

// GenerateImageComponentV2 returns top-level image component from embedded component.
func GenerateImageComponentV2(os string, image *storage.ImageV2, index int, from *storage.EmbeddedImageScanComponent) (*storage.ImageComponentV2, error) {
	componentID := scancomponent.ComponentIDV2(from, image.GetId(), index)

	ret := &storage.ImageComponentV2{
		Id:              componentID,
		Name:            from.GetName(),
		Version:         from.GetVersion(),
		Source:          from.GetSource(),
		FixedBy:         from.GetFixedBy(),
		RiskScore:       from.GetRiskScore(),
		Priority:        from.GetPriority(),
		OperatingSystem: os,
		ImageIdV2:       image.GetId(),
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
