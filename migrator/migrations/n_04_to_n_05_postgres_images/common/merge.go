package common

import (
	"sort"

	"github.com/stackrox/rox/generated/storage"
	converter "github.com/stackrox/rox/migrator/migrations/cvehelper"
	"github.com/stackrox/rox/pkg/dackbox/edges"
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
		// Parse the IDs of the edge.
		imageComponentEdgeID, err := edges.FromString(cp.Edge.GetId())
		if err != nil {
			log.Error(err)
			continue
		}

		if imageComponentEdgeID.ParentID != image.GetId() {
			log.Error("image to component edge does not match image")
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

func generateEmbeddedComponent(os string, cp ComponentParts, imageCVEEdges map[string]*storage.ImageCVEEdge) *storage.EmbeddedImageScanComponent {
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
		cveEdge := imageCVEEdges[cve.Cve.GetId()]
		// This is due to the scenario when the CVE was never found in the image, but instead
		// the <component, version> tuple was found in another image that may have had these specific vulns.
		// When getting the image, we should filter these vulns out for correctness. Note, this does not
		// fix what will happen in the UI
		if cveEdge == nil {
			continue
		}
		ret.Vulns = append(ret.Vulns, generateEmbeddedCVE(os, cve, imageCVEEdges[cve.Cve.GetId()]))
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
	if cp.Cve == nil || cp.Edge == nil {
		return nil
	}

	ret := converter.ProtoCVEToEmbeddedCVE(cp.Cve)
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

	if distroSpecifics, ok := cp.Cve.GetDistroSpecifics()[os]; ok {
		ret.Severity = distroSpecifics.GetSeverity()
		ret.Cvss = distroSpecifics.GetCvss()
		ret.CvssV2 = distroSpecifics.GetCvssV2()
		ret.CvssV3 = distroSpecifics.GetCvssV3()
		ret.ScoreVersion = cveScoreVersionToEmbeddedScoreVersion(distroSpecifics.GetScoreVersion())
	}

	return ret
}
