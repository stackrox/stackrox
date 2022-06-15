package common

import (
	"sort"

	"github.com/stackrox/rox/central/cve/converter"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search/postgres"
)

// Merge merges the images parts into an image.
func Merge(parts ImageParts) *storage.Image {
	ret := parts.Image.Clone()
	mergeComponents(parts, ret)
	return ret
}

func mergeComponents(parts ImageParts, image *storage.Image) {
	// If the image has a nil scan, there is nothing to fill in.
	if image.Scan == nil {
		return
	}

	// Use the edges to combine into the parent image.
	for _, cp := range parts.Children {
		var imageIDFromEdgeID string
		IDParts := postgres.IDToParts(cp.Edge.GetId())
		if len(IDParts) == 0 {
			log.Error("image to component edge does not have primary keys")
			continue
		}
		imageIDFromEdgeID = IDParts[0]
		if imageIDFromEdgeID != image.GetId() {
			log.Error("image to component edge does not match image")
			continue
		}
		// Generate an embedded component for the edge and non-embedded version.
		image.Scan.Components = append(image.Scan.Components, generateEmbeddedComponent(cp, parts.ImageCVEEdges))
	}

	components := image.GetScan().GetComponents()
	sort.SliceStable(components, func(i, j int) bool {
		return components[i].GetName() < components[j].GetName()
	})
	for _, comp := range components {
		sort.SliceStable(comp.Vulns, func(i, j int) bool {
			return comp.Vulns[i].GetCve() < comp.Vulns[j].GetCve()
		})
	}
}

func generateEmbeddedComponent(cp ComponentParts, imageCVEEdges map[string]*storage.ImageCVEEdge) *storage.EmbeddedImageScanComponent {
	if cp.Component == nil || cp.Edge == nil {
		return nil
	}
	ret := &storage.EmbeddedImageScanComponent{
		Name:      cp.Component.GetName(),
		Version:   cp.Component.GetVersion(),
		License:   cp.Component.GetLicense().Clone(),
		Source:    cp.Component.GetSource(),
		Location:  cp.Edge.GetLocation(),
		FixedBy:   cp.Component.GetFixedBy(),
		RiskScore: cp.Component.GetRiskScore(),
		Priority:  cp.Component.GetPriority(),
	}

	if cp.Edge.HasLayerIndex != nil {
		ret.HasLayerIndex = &storage.EmbeddedImageScanComponent_LayerIndex{
			LayerIndex: cp.Edge.GetLayerIndex(),
		}
	}

	if cp.Component.GetSetTopCvss() != nil {
		ret.SetTopCvss = &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: cp.Component.GetTopCvss()}
	}

	ret.Vulns = make([]*storage.EmbeddedVulnerability, 0, len(cp.Children))
	for _, cve := range cp.Children {
		ret.Vulns = append(ret.Vulns, generateEmbeddedCVE(cve, imageCVEEdges[cve.CVE.GetId()]))
	}
	return ret
}

func generateEmbeddedCVE(cp CVEParts, imageCVEEdge *storage.ImageCVEEdge) *storage.EmbeddedVulnerability {
	if cp.CVE == nil || cp.Edge == nil {
		return nil
	}

	ret := converter.ImageCVEToEmbeddedVulnerability(cp.CVE)
	if cp.Edge.IsFixable {
		ret.SetFixedBy = &storage.EmbeddedVulnerability_FixedBy{
			FixedBy: cp.Edge.GetFixedBy(),
		}
	}
	ret.FirstImageOccurrence = imageCVEEdge.GetFirstImageOccurrence()
	if state := imageCVEEdge.GetState(); state != storage.VulnerabilityState_OBSERVED {
		ret.State = state
	}
	return ret
}
