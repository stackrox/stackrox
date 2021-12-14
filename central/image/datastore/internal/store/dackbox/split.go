package dackbox

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
		image: image.Clone(),
	}

	// These need to be called in order.
	parts.listImage = splitListImage(parts)
	if withComponents {
		parts.children = splitComponents(parts)
	}
	parts.imageCVEEdges = generateImageToCVEEdges(parts)

	// Clear components in the top level image.
	if parts.image.GetScan() != nil {
		parts.image.Scan.Components = nil
	}

	return parts
}

func splitListImage(parts ImageParts) *storage.ListImage {
	return types.ConvertImageToListImage(parts.image)
}

func splitComponents(parts ImageParts) []ComponentParts {
	ret := make([]ComponentParts, 0, len(parts.image.GetScan().GetComponents()))
	for _, component := range parts.image.GetScan().GetComponents() {
		cp := ComponentParts{}
		cp.component = generateImageComponent(component)
		cp.edge = generateImageComponentEdge(parts.image, cp.component, component)
		cp.children = splitCVEs(parts.image.GetScan().GetOperatingSystem(), cp, component)

		ret = append(ret, cp)
	}

	return ret
}

func splitCVEs(os string, component ComponentParts, embedded *storage.EmbeddedImageScanComponent) []CVEParts {
	ret := make([]CVEParts, 0, len(embedded.GetVulns()))
	for _, cve := range embedded.GetVulns() {
		cp := CVEParts{}
		cp.cve = converter.EmbeddedCVEToProtoCVE(os, cve)
		cp.edge = generateComponentCVEEdge(component.component, cp.cve, cve)

		ret = append(ret, cp)
	}

	return ret
}

func generateComponentCVEEdge(convertedComponent *storage.ImageComponent, convertedCVE *storage.CVE, embedded *storage.EmbeddedVulnerability) *storage.ComponentCVEEdge {
	ret := &storage.ComponentCVEEdge{
		Id:        edges.EdgeID{ParentID: convertedComponent.GetId(), ChildID: convertedCVE.GetId()}.ToString(),
		IsFixable: embedded.GetFixedBy() != "",
	}
	if ret.IsFixable {
		ret.HasFixedBy = &storage.ComponentCVEEdge_FixedBy{
			FixedBy: embedded.GetFixedBy(),
		}
	}
	return ret
}

func generateImageComponent(from *storage.EmbeddedImageScanComponent) *storage.ImageComponent {
	ret := &storage.ImageComponent{
		Id:        scancomponent.ComponentID(from.GetName(), from.GetVersion()),
		Name:      from.GetName(),
		Version:   from.GetVersion(),
		License:   from.GetLicense().Clone(),
		Source:    from.GetSource(),
		FixedBy:   from.GetFixedBy(),
		RiskScore: from.GetRiskScore(),
	}

	if from.GetSetTopCvss() != nil {
		ret.SetTopCvss = &storage.ImageComponent_TopCvss{TopCvss: from.GetTopCvss()}
	}
	return ret
}

func generateImageComponentEdge(image *storage.Image, converted *storage.ImageComponent, embedded *storage.EmbeddedImageScanComponent) *storage.ImageComponentEdge {
	ret := &storage.ImageComponentEdge{
		Id: edges.EdgeID{ParentID: image.GetId(), ChildID: converted.GetId()}.ToString(),
	}
	if embedded.HasLayerIndex != nil {
		ret.HasLayerIndex = &storage.ImageComponentEdge_LayerIndex{
			LayerIndex: embedded.GetLayerIndex(),
		}
	}
	ret.Location = embedded.GetLocation()
	return ret
}

func generateImageToCVEEdges(parts ImageParts) map[string]*storage.ImageCVEEdge {
	imageCVEEdges := make(map[string]*storage.ImageCVEEdge)
	for _, componentParts := range parts.children {
		for _, cveParts := range componentParts.children {
			if _, ok := imageCVEEdges[cveParts.cve.GetId()]; !ok {
				imageCVEEdges[cveParts.cve.GetId()] = &storage.ImageCVEEdge{
					Id:    edges.EdgeID{ParentID: parts.image.GetId(), ChildID: cveParts.cve.GetId()}.ToString(),
					State: getVulnState(cveParts.cve),
				}

			}
		}
	}
	return imageCVEEdges
}

func getVulnState(cve *storage.CVE) storage.VulnerabilityState {
	if cve.GetSuppressed() {
		return storage.VulnerabilityState_DEFERRED
	}
	return storage.VulnerabilityState_OBSERVED
}
