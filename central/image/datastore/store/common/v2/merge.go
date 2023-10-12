package common

import (
	"sort"

	"github.com/stackrox/rox/central/cve/converter/utils"
	"github.com/stackrox/rox/generated/storage"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
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
		IDParts := pgSearch.IDToParts(cp.Edge.GetId())
		if len(IDParts) == 0 {
			log.Error("image to component edge does not have primary keys")
			continue
		}
		imageIDFromEdgeID := IDParts[0]

		if imageIDFromEdgeID != image.GetId() {
			log.Error("image to component edge does not match image")
			continue
		}
		if cp.Component == nil || cp.Edge == nil {
			log.Errorf("UNEXPECTED: nil component or edge when retrieving components for image %q", image.GetId())
			continue
		}
		// Generate an embedded component for the edge and non-embedded version.
		image.Scan.Components = append(image.Scan.Components, generateEmbeddedComponent(image.GetScan().GetOperatingSystem(), cp, parts.ImageCVEEdges))
	}

	sort.SliceStable(image.GetScan().GetComponents(), func(i, j int) bool {
		return image.GetScan().GetComponents()[i].GetName() < image.GetScan().GetComponents()[j].GetName()
	})
	for _, comp := range image.GetScan().GetComponents() {
		sort.SliceStable(comp.Vulns, func(i, j int) bool {
			return comp.Vulns[i].GetCve() < comp.Vulns[j].GetCve()
		})
	}
}

func generateEmbeddedComponent(_ string, cp ComponentParts, imageCVEEdges map[string]*storage.ImageCVEEdge) *storage.EmbeddedImageScanComponent {
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
		cveEdge := imageCVEEdges[cve.CVE.GetId()]
		// This is due to the scenario when the CVE was never found in the image, but instead
		// the <component, version> tuple was found in another image that may have had these specific vulns.
		// When getting the image, we should filter these vulns out for correctness. Note, this does not
		// fix what will happen in the UI
		if cveEdge == nil {
			continue
		}
		if cve.CVE == nil || cve.Edge == nil {
			log.Errorf("UNEXPECTED: nil cve or edge when retrieving cves for component %q", cp.Component.GetId())
			continue
		}
		ret.Vulns = append(ret.Vulns, generateEmbeddedCVE(cve, imageCVEEdges[cve.CVE.GetId()]))
	}
	return ret
}

func generateEmbeddedCVE(cp CVEParts, imageCVEEdge *storage.ImageCVEEdge) *storage.EmbeddedVulnerability {
	ret := utils.ImageCVEToEmbeddedVulnerability(cp.CVE)
	if cp.Edge.IsFixable {
		ret.SetFixedBy = &storage.EmbeddedVulnerability_FixedBy{
			FixedBy: cp.Edge.GetFixedBy(),
		}
	}
	ret.FirstImageOccurrence = imageCVEEdge.GetFirstImageOccurrence()

	// The `Suppressed` field is transferred to `State` field (as DEFERRED) in `converter.ProtoCVEToEmbeddedCVE`.
	// Now visit image-cve edge to derive the state.
	if state := imageCVEEdge.GetState(); state != storage.VulnerabilityState_OBSERVED {
		ret.State = state
	}
	return ret
}
