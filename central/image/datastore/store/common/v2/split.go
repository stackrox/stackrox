package common

import (
	"github.com/stackrox/rox/central/cve/converter/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scancomponent"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/set"
)

// TODO(ROX-28123): Remove file

// Split splits the input image into a set of parts.
func Split(image *storage.Image, withComponents bool) (ImageParts, error) {
	parts := ImageParts{
		Image:         image.CloneVT(),
		ImageCVEEdges: make(map[string]*storage.ImageCVEEdge),
	}

	if withComponents {
		parts.Children = splitComponents(parts)
	}

	// Clear components in the top level image.
	if parts.Image.GetScan() != nil {
		parts.Image.GetScan().SetComponents(nil)
	}
	return parts, nil
}

func splitComponents(parts ImageParts) []ComponentParts {
	ret := make([]ComponentParts, 0, len(parts.Image.GetScan().GetComponents()))
	addedComponents := set.NewStringSet()
	for _, component := range parts.Image.GetScan().GetComponents() {
		generatedComponent := GenerateImageComponent(parts.Image.GetScan().GetOperatingSystem(), component)
		if !addedComponents.Add(generatedComponent.GetId()) {
			continue
		}

		cp := ComponentParts{}
		cp.Component = generatedComponent
		cp.Edge = generateImageComponentEdge(parts.Image, cp.Component, component)
		cp.Children = splitCVEs(parts, cp, component)

		ret = append(ret, cp)
	}

	return ret
}

func splitCVEs(parts ImageParts, component ComponentParts, embedded *storage.EmbeddedImageScanComponent) []CVEParts {
	ret := make([]CVEParts, 0, len(embedded.GetVulns()))
	addedCVEs := set.NewStringSet()
	for _, cve := range embedded.GetVulns() {
		convertedCVE := utils.EmbeddedVulnerabilityToImageCVE(parts.Image.GetScan().GetOperatingSystem(), cve)
		if !addedCVEs.Add(convertedCVE.GetId()) {
			continue
		}
		cp := CVEParts{}
		cp.CVE = convertedCVE
		cp.Edge = generateComponentCVEEdge(component.Component, cp.CVE, cve)
		if _, ok := parts.ImageCVEEdges[cp.CVE.GetId()]; !ok {
			parts.ImageCVEEdges[cp.CVE.GetId()] = generateImageCVEEdge(parts.Image.GetId(), cp.CVE, cve)
		}
		ret = append(ret, cp)
	}

	return ret
}

func generateComponentCVEEdge(convertedComponent *storage.ImageComponent, convertedCVE *storage.ImageCVE, embedded *storage.EmbeddedVulnerability) *storage.ComponentCVEEdge {
	ret := &storage.ComponentCVEEdge{}
	ret.SetId(pgSearch.IDFromPks([]string{convertedComponent.GetId(), convertedCVE.GetId()}))
	ret.SetIsFixable(embedded.GetFixedBy() != "")
	ret.SetImageCveId(convertedCVE.GetId())
	ret.SetImageComponentId(convertedComponent.GetId())

	if ret.GetIsFixable() {
		ret.SetFixedBy(embedded.GetFixedBy())
	}
	return ret
}

// Deprecated: replaced with equivalent functions using storage.ImageComponentV2
// GenerateImageComponent returns top-level image component from embedded component.
func GenerateImageComponent(os string, from *storage.EmbeddedImageScanComponent) *storage.ImageComponent {
	ret := &storage.ImageComponent{}
	ret.SetId(scancomponent.ComponentID(from.GetName(), from.GetVersion(), os))
	ret.SetName(from.GetName())
	ret.SetVersion(from.GetVersion())
	ret.SetLicense(from.GetLicense().CloneVT())
	ret.SetSource(from.GetSource())
	ret.SetFixedBy(from.GetFixedBy())
	ret.SetRiskScore(from.GetRiskScore())
	ret.SetPriority(from.GetPriority())
	ret.SetOperatingSystem(os)

	if from.GetSetTopCvss() != nil {
		ret.Set_TopCvss(from.GetTopCvss())
	}
	return ret
}

func generateImageComponentEdge(image *storage.Image, convImgComponent *storage.ImageComponent, embedded *storage.EmbeddedImageScanComponent) *storage.ImageComponentEdge {
	ret := &storage.ImageComponentEdge{}
	ret.SetId(pgSearch.IDFromPks([]string{image.GetId(), convImgComponent.GetId()}))
	ret.SetImageId(image.GetId())
	ret.SetImageComponentId(convImgComponent.GetId())
	ret.SetLocation(embedded.GetLocation())

	if embedded.HasHasLayerIndex() {
		ret.SetLayerIndex(embedded.GetLayerIndex())
	}
	return ret
}

func generateImageCVEEdge(imageID string, convertedCVE *storage.ImageCVE, embedded *storage.EmbeddedVulnerability) *storage.ImageCVEEdge {
	ret := &storage.ImageCVEEdge{}
	ret.SetId(pgSearch.IDFromPks([]string{imageID, convertedCVE.GetId()}))
	ret.SetState(embedded.GetState())
	ret.SetImageId(imageID)
	ret.SetImageCveId(convertedCVE.GetId())
	return ret
}
