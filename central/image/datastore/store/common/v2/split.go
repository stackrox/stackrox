package common

import (
	"github.com/stackrox/rox/central/cve/converter/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scancomponent"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/set"
)

// Split splits the input image into a set of parts.
func Split(image *storage.Image, withComponents bool) ImageParts {
	parts := ImageParts{
		Image:         image.Clone(),
		ImageCVEEdges: make(map[string]*storage.ImageCVEEdge),
	}

	// These need to be called in order.
	if withComponents {
		parts.Children = splitComponents(parts)
	}

	// Clear components in the top level image.
	if parts.Image.GetScan() != nil {
		parts.Image.Scan.Components = nil
	}
	return parts
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
	ret := &storage.ComponentCVEEdge{
		Id:               pgSearch.IDFromPks([]string{convertedComponent.GetId(), convertedCVE.GetId()}),
		IsFixable:        embedded.GetFixedBy() != "",
		ImageCveId:       convertedCVE.GetId(),
		ImageComponentId: convertedComponent.GetId(),
	}

	if ret.IsFixable {
		ret.HasFixedBy = &storage.ComponentCVEEdge_FixedBy{
			FixedBy: embedded.GetFixedBy(),
		}
	}
	return ret
}

// GenerateImageComponent returns top-level image component from embedded component.
func GenerateImageComponent(os string, from *storage.EmbeddedImageScanComponent) *storage.ImageComponent {
	ret := &storage.ImageComponent{
		Id:              scancomponent.ComponentID(from.GetName(), from.GetVersion(), os),
		Name:            from.GetName(),
		Version:         from.GetVersion(),
		License:         from.GetLicense().Clone(),
		Source:          from.GetSource(),
		FixedBy:         from.GetFixedBy(),
		RiskScore:       from.GetRiskScore(),
		Priority:        from.GetPriority(),
		OperatingSystem: os,
	}

	if from.GetSetTopCvss() != nil {
		ret.SetTopCvss = &storage.ImageComponent_TopCvss{TopCvss: from.GetTopCvss()}
	}
	return ret
}

func generateImageComponentEdge(image *storage.Image, convImgComponent *storage.ImageComponent, embedded *storage.EmbeddedImageScanComponent) *storage.ImageComponentEdge {
	ret := &storage.ImageComponentEdge{
		Id:               pgSearch.IDFromPks([]string{image.GetId(), convImgComponent.GetId()}),
		ImageId:          image.GetId(),
		ImageComponentId: convImgComponent.GetId(),
		Location:         embedded.GetLocation(),
	}

	if embedded.HasLayerIndex != nil {
		ret.HasLayerIndex = &storage.ImageComponentEdge_LayerIndex{
			LayerIndex: embedded.GetLayerIndex(),
		}
	}
	return ret
}

func generateImageCVEEdge(imageID string, convertedCVE *storage.ImageCVE, embedded *storage.EmbeddedVulnerability) *storage.ImageCVEEdge {
	ret := &storage.ImageCVEEdge{
		Id:         pgSearch.IDFromPks([]string{imageID, convertedCVE.GetId()}),
		State:      embedded.GetState(),
		ImageId:    imageID,
		ImageCveId: convertedCVE.GetId(),
	}
	return ret
}
