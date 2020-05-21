package dackbox

import (
	"github.com/stackrox/rox/central/cve/converter"
	"github.com/stackrox/rox/central/imagecomponent"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stackrox/rox/pkg/images/types"
)

// Split splits the input image into a set of parts.
func Split(image *storage.Image) ImageParts {
	parts := ImageParts{
		image: image.Clone(),
	}

	// These need to be called in order.
	parts.listImage = splitListImage(parts)
	parts.children = splitComponents(parts)

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
		cp.children = splitCVEs(cp, component)

		ret = append(ret, cp)
	}

	return ret
}

func splitCVEs(component ComponentParts, embedded *storage.EmbeddedImageScanComponent) []CVEParts {
	ret := make([]CVEParts, 0, len(embedded.GetVulns()))
	for _, cve := range embedded.GetVulns() {
		cp := CVEParts{}
		cp.cve = converter.EmbeddedCVEToProtoCVE(cve)
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
		Id:        imagecomponent.ComponentID{Name: from.GetName(), Version: from.GetVersion()}.ToString(),
		Name:      from.GetName(),
		Version:   from.GetVersion(),
		License:   from.GetLicense().Clone(),
		Source:    from.GetSource(),
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
