package common

import (
	"github.com/stackrox/rox/central/cve/converter/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/scancomponent"
)

// SplitV2 splits the input image into a set of parts.
func SplitV2(image *storage.Image, withComponents bool) (ImageParts, error) {
	if !features.FlattenCVEData.Enabled() {
		return ImageParts{}, nil
	}
	parts := ImageParts{
		Image:         image.CloneVT(),
		ImageCVEEdges: make(map[string]*storage.ImageCVEEdge),
	}

	if withComponents {
		var err error
		parts.Children, err = splitComponentsV2(parts)
		if err != nil {
			return ImageParts{}, err
		}
	}

	// Clear components in the top level image.
	if parts.Image.GetScan() != nil {
		parts.Image.Scan.Components = nil
	}
	return parts, nil
}

func splitComponentsV2(parts ImageParts) ([]ComponentParts, error) {
	ret := make([]ComponentParts, 0, len(parts.Image.GetScan().GetComponents()))
	componentMap := make(map[string]*storage.EmbeddedImageScanComponent)
	for _, component := range parts.Image.GetScan().GetComponents() {
		generatedComponentV2, err := GenerateImageComponentV2(parts.Image.GetScan().GetOperatingSystem(), parts.Image, component)
		if err != nil {
			return nil, err
		}

		// dedupe components within the component
		if _, ok := componentMap[generatedComponentV2.GetId()]; ok {
			log.Infof("Component %s-%s has already been processed in the image. Skipping...", component.GetName(), component.GetVersion())
			continue
		}

		componentMap[generatedComponentV2.GetId()] = component

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
	cveMap := make(map[string]*storage.EmbeddedVulnerability)
	for _, cve := range embedded.GetVulns() {
		convertedCVE, err := utils.EmbeddedVulnerabilityToImageCVEV2(imageID, componentID, cve)
		if err != nil {
			return nil, err
		}

		// dedupe CVEs within the component
		if _, ok := cveMap[convertedCVE.GetId()]; ok {
			log.Infof("CVE %s has already been processed in the image. Skipping...", cve.GetCve())
			continue
		}

		cveMap[convertedCVE.GetId()] = cve

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
func GenerateImageComponentV2FromImageV2(os string, image *storage.ImageV2, from *storage.EmbeddedImageScanComponent) (*storage.ImageComponentV2, error) {
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
