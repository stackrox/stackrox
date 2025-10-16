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
		parts.Image.GetScan().SetComponents(nil)
	}
	return parts, nil
}

func splitComponents(parts ImagePartsV2) ([]ComponentPartsV2, error) {
	ret := make([]ComponentPartsV2, 0, len(parts.Image.GetScan().GetComponents()))
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

		cp := CVEPartsV2{
			CVEV2: convertedCVE,
		}
		ret = append(ret, cp)
	}

	return ret, nil
}

// GenerateImageComponentV2 returns top-level image component from embedded component.
func GenerateImageComponentV2(os string, image *storage.ImageV2, from *storage.EmbeddedImageScanComponent) (*storage.ImageComponentV2, error) {
	componentID, err := scancomponent.ComponentIDV2(from, image.GetId())
	if err != nil {
		return nil, err
	}

	ret := &storage.ImageComponentV2{}
	ret.SetId(componentID)
	ret.SetName(from.GetName())
	ret.SetVersion(from.GetVersion())
	ret.SetSource(from.GetSource())
	ret.SetFixedBy(from.GetFixedBy())
	ret.SetRiskScore(from.GetRiskScore())
	ret.SetPriority(from.GetPriority())
	ret.SetOperatingSystem(os)
	ret.SetImageIdV2(image.GetId())
	ret.SetLocation(from.GetLocation())
	ret.SetArchitecture(from.GetArchitecture())

	if from.GetSetTopCvss() != nil {
		ret.Set_TopCvss(from.GetTopCvss())
	}

	if from.HasHasLayerIndex() {
		ret.Set_LayerIndex(from.GetLayerIndex())
	}

	return ret, nil
}
