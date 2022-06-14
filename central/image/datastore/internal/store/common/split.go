package common

import (
	"github.com/stackrox/stackrox/central/cve/converter"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/edges"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/images/types"
	"github.com/stackrox/stackrox/pkg/scancomponent"
	"github.com/stackrox/stackrox/pkg/search/postgres"
	"github.com/stackrox/stackrox/pkg/set"
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
	addedComponents := set.NewStringSet()
	for _, component := range parts.Image.GetScan().GetComponents() {
		generatedComponent := generateImageComponent(parts.Image.GetScan().GetOperatingSystem(), component)
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
		convertedCVE := converter.EmbeddedCVEToProtoCVE(parts.Image.GetScan().GetOperatingSystem(), cve)
		if !addedCVEs.Add(convertedCVE.GetId()) {
			continue
		}
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
		IsFixable:        embedded.GetFixedBy() != "",
		ImageCveId:       convertedCVE.GetId(),
		ImageComponentId: convertedComponent.GetId(),
	}
	if features.PostgresDatastore.Enabled() {
		ret.Id = postgres.IDFromPks([]string{convertedComponent.GetId(), convertedCVE.GetId()})
	} else {
		ret.Id = edges.EdgeID{ParentID: convertedComponent.GetId(), ChildID: convertedCVE.GetId()}.ToString()
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
		ImageId:          image.GetId(),
		ImageComponentId: convImgComponent.GetId(),
		Location:         embedded.GetLocation(),
	}

	if features.PostgresDatastore.Enabled() {
		ret.Id = postgres.IDFromPks([]string{image.GetId(), convImgComponent.GetId()})
	} else {
		ret.Id = edges.EdgeID{ParentID: image.GetId(), ChildID: convImgComponent.GetId()}.ToString()
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
		State:      embedded.GetState(),
		ImageId:    imageID,
		ImageCveId: convertedCVE.GetId(),
	}

	if features.PostgresDatastore.Enabled() {
		ret.Id = postgres.IDFromPks([]string{imageID, convertedCVE.GetId()})
	} else {
		ret.Id = edges.EdgeID{ParentID: imageID, ChildID: convertedCVE.GetId()}.ToString()
	}

	if ret.GetState() != storage.VulnerabilityState_OBSERVED {
		return ret
	}
	return ret
}
