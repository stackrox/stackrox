package common

import (
	"github.com/stackrox/rox/central/cve/converter"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/scancomponent"
)

// Split splits the input image into a set of parts.
func Split(image *storage.Image, withComponents bool) ImageParts {
	parts := ImageParts{
		Image:         image.Clone(),
		ImageCVEEdges: make(map[string]*storage.ImageCVEEdge),
	}

	// These need to be called in order.
	parts.ListImage = splitListImage(parts)
	if withComponents {
		parts.Children = splitComponents(parts)
	}

	// Clear components in the top level image.
	if parts.Image.GetScan() != nil {
		parts.Image.Scan.Components = nil
	}
	return parts
}

func splitListImage(parts ImageParts) *storage.ListImage {
	return types.ConvertImageToListImage(parts.Image)
}

func splitComponents(parts ImageParts) []ComponentParts {
	ret := make([]ComponentParts, 0, len(parts.Image.GetScan().GetComponents()))
	for _, component := range parts.Image.GetScan().GetComponents() {
		cp := ComponentParts{}
		cp.Component = generateImageComponent(parts.Image.GetScan().GetOperatingSystem(), component)
		cp.Edge = generateImageComponentEdge(parts.Image, cp.Component, component)
		cp.Children = splitCVEs(parts, cp, component)

		ret = append(ret, cp)
	}

	return ret
}

func splitCVEs(parts ImageParts, component ComponentParts, embedded *storage.EmbeddedImageScanComponent) []CVEParts {
	ret := make([]CVEParts, 0, len(embedded.GetVulns()))
	for _, cve := range embedded.GetVulns() {
		cp := CVEParts{}
		cp.Cve = converter.EmbeddedCVEToProtoCVE(parts.Image.GetScan().GetOperatingSystem(), cve)
		cp.Edge = generateComponentCVEEdge(component.Component, cp.Cve, cve)
		if _, ok := parts.ImageCVEEdges[cp.Cve.GetId()]; !ok {
			parts.ImageCVEEdges[cp.Cve.GetId()] = generateImageCVEEdge(parts.Image.GetId(), cp.Cve, cve)
		}
		ret = append(ret, cp)
	}

	return ret
}

func generateComponentCVEEdge(convertedComponent *storage.ImageComponent, convertedCVE *storage.CVE, embedded *storage.EmbeddedVulnerability) *storage.ComponentCVEEdge {
	ret := &storage.ComponentCVEEdge{
		Id:                 edges.EdgeID{ParentID: convertedComponent.GetId(), ChildID: convertedCVE.GetId()}.ToString(),
		IsFixable:          embedded.GetFixedBy() != "",
		CveId:              convertedCVE.GetId(),
		ImageComponentId:   convertedComponent.GetId(),
		CveOperatingSystem: convertedComponent.GetOperatingSystem(),
	}

	if ret.IsFixable {
		ret.HasFixedBy = &storage.ComponentCVEEdge_FixedBy{
			FixedBy: embedded.GetFixedBy(),
		}
	}
	return ret
}

func generateImageComponent(os string, from *storage.EmbeddedImageScanComponent) *storage.ImageComponent {
	ret := &storage.ImageComponent{
		Id:              scancomponent.ComponentID(from.GetName(), from.GetVersion(), os),
		Name:            from.GetName(),
		Version:         from.GetVersion(),
		License:         from.GetLicense().Clone(),
		Source:          from.GetSource(),
		FixedBy:         from.GetFixedBy(),
		RiskScore:       from.GetRiskScore(),
		OperatingSystem: os,
	}

	if from.GetSetTopCvss() != nil {
		ret.SetTopCvss = &storage.ImageComponent_TopCvss{TopCvss: from.GetTopCvss()}
	}
	return ret
}

func generateImageComponentEdge(image *storage.Image, convImgComponent *storage.ImageComponent, embedded *storage.EmbeddedImageScanComponent) *storage.ImageComponentEdge {
	ret := &storage.ImageComponentEdge{
		Id:               edges.EdgeID{ParentID: image.GetId(), ChildID: convImgComponent.GetId()}.ToString(),
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

func generateImageCVEEdge(imageID string, convertedCVE *storage.CVE, embedded *storage.EmbeddedVulnerability) *storage.ImageCVEEdge {
	ret := &storage.ImageCVEEdge{
		Id:    edges.EdgeID{ParentID: imageID, ChildID: convertedCVE.GetId()}.ToString(),
		State: embedded.GetState(),
	}
	if ret.GetState() != storage.VulnerabilityState_OBSERVED {
		return ret
	}
	return ret
}
