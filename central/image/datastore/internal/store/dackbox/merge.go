package dackbox

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/cve/converter"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/edges"
)

// Merge merges the images parts into an image.
func Merge(parts ImageParts) *storage.Image {
	ret := proto.Clone(parts.image).(*storage.Image)
	mergeComponents(parts, ret)
	return ret
}

func mergeComponents(parts ImageParts, image *storage.Image) {
	// If the image has a nil scan, there is nothing to fill in.
	if image.Scan == nil {
		return
	}

	// Use the edges to combine into the parent image.
	for _, cp := range parts.children {
		// Parse the IDs of the edge.
		imageComponentEdgeIDs, err := edges.FromString(cp.edge.GetId())
		if err != nil {
			log.Error(err)
			continue
		}
		if imageComponentEdgeIDs.ParentID != image.GetId() {
			log.Error("image to component edge does not match image")
			continue
		}

		// Generate an embedded component for the edge and non-embedded version.
		image.Scan.Components = append(image.Scan.Components, generateEmbeddedComponent(cp))
	}
}

func generateEmbeddedComponent(cp ComponentParts) *storage.EmbeddedImageScanComponent {
	if cp.component == nil || cp.edge == nil {
		return nil
	}
	ret := &storage.EmbeddedImageScanComponent{
		Name:     cp.component.GetName(),
		Version:  cp.component.GetVersion(),
		License:  proto.Clone(cp.component.GetLicense()).(*storage.License),
		Source:   cp.component.GetSource(),
		Location: cp.edge.GetLocation(),
	}
	if cp.edge.HasLayerIndex != nil {
		ret.HasLayerIndex = &storage.EmbeddedImageScanComponent_LayerIndex{
			LayerIndex: cp.edge.GetLayerIndex(),
		}
	}

	ret.Vulns = make([]*storage.EmbeddedVulnerability, 0, len(cp.children))
	for _, cve := range cp.children {
		ret.Vulns = append(ret.Vulns, generateEmbeddedCVE(cve))
	}
	return ret
}

func generateEmbeddedCVE(cp CVEParts) *storage.EmbeddedVulnerability {
	if cp.cve == nil || cp.edge == nil {
		return nil
	}

	ret := converter.ProtoCVEToEmbeddedCVE(cp.cve)
	if cp.edge.IsFixable {
		ret.SetFixedBy = &storage.EmbeddedVulnerability_FixedBy{
			FixedBy: cp.edge.GetFixedBy(),
		}
	}
	return ret
}
