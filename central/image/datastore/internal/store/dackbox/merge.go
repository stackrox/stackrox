package dackbox

import (
	"github.com/stackrox/rox/central/cve/converter"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/edges"
)

// Merge merges the images parts into an image.
func Merge(parts ImageParts) *storage.Image {
	ret := parts.image.Clone()
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
		image.Scan.Components = append(image.Scan.Components, generateEmbeddedComponent(image.GetScan().GetOperatingSystem(), cp, parts.imageCVEEdges))
	}
}

func generateEmbeddedComponent(os string, cp ComponentParts, imageCVEEdges map[string]*storage.ImageCVEEdge) *storage.EmbeddedImageScanComponent {
	if cp.component == nil || cp.edge == nil {
		return nil
	}
	ret := &storage.EmbeddedImageScanComponent{
		Name:      cp.component.GetName(),
		Version:   cp.component.GetVersion(),
		License:   cp.component.GetLicense().Clone(),
		Source:    cp.component.GetSource(),
		Location:  cp.edge.GetLocation(),
		FixedBy:   cp.component.GetFixedBy(),
		RiskScore: cp.component.GetRiskScore(),
	}

	if cp.edge.HasLayerIndex != nil {
		ret.HasLayerIndex = &storage.EmbeddedImageScanComponent_LayerIndex{
			LayerIndex: cp.edge.GetLayerIndex(),
		}
	}

	if cp.component.GetSetTopCvss() != nil {
		ret.SetTopCvss = &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: cp.component.GetTopCvss()}
	}

	ret.Vulns = make([]*storage.EmbeddedVulnerability, 0, len(cp.children))
	for _, cve := range cp.children {
		cveEdge := imageCVEEdges[cve.cve.GetId()]
		// This is due to the scenario when the CVE was never found in the image, but instead
		// the <component, version> tuple was found in another image that may have had these specific vulns.
		// When getting the image, we should filter these vulns out for correctness. Note, this does not
		// fix what will happen in the UI
		if cveEdge.GetFirstImageOccurrence() == nil {
			continue
		}
		ret.Vulns = append(ret.Vulns, generateEmbeddedCVE(os, cve, imageCVEEdges[cve.cve.GetId()]))
	}
	return ret
}

func cveScoreVersionToEmbeddedScoreVersion(v storage.CVE_ScoreVersion) storage.EmbeddedVulnerability_ScoreVersion {
	switch v {
	case storage.CVE_V2:
		return storage.EmbeddedVulnerability_V2
	case storage.CVE_V3:
		return storage.EmbeddedVulnerability_V3
	default:
		return storage.EmbeddedVulnerability_V2
	}
}

func generateEmbeddedCVE(os string, cp CVEParts, imageCVEEdge *storage.ImageCVEEdge) *storage.EmbeddedVulnerability {
	if cp.cve == nil || cp.edge == nil {
		return nil
	}

	ret := converter.ProtoCVEToEmbeddedCVE(cp.cve)
	if cp.edge.IsFixable {
		ret.SetFixedBy = &storage.EmbeddedVulnerability_FixedBy{
			FixedBy: cp.edge.GetFixedBy(),
		}
	}
	ret.FirstImageOccurrence = imageCVEEdge.GetFirstImageOccurrence()

	// The `Suppressed` field is transferred to `State` field (as DEFERRED) in `converter.ProtoCVEToEmbeddedCVE`.
	// Now visit image-cve edge to derive the state.
	if state := imageCVEEdge.GetState(); state != storage.VulnerabilityState_OBSERVED {
		ret.State = state
	}

	if distroSpecifics, ok := cp.cve.GetDistroSpecifics()[os]; ok {
		ret.Severity = distroSpecifics.GetSeverity()
		ret.Cvss = distroSpecifics.GetCvss()
		ret.CvssV2 = distroSpecifics.GetCvssV2()
		ret.CvssV3 = distroSpecifics.GetCvssV3()
		ret.ScoreVersion = cveScoreVersionToEmbeddedScoreVersion(distroSpecifics.GetScoreVersion())
	}

	return ret
}
