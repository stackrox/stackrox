package common

import (
	"github.com/stackrox/rox/central/cve/converter/utils"
	"github.com/stackrox/rox/generated/storage"
)

// MergeV2 merges the images parts into an image.
func MergeV2(parts ImageParts) *storage.Image {
	mergeComponentsV2(parts, parts.Image)
	return parts.Image
}

func mergeComponentsV2(parts ImageParts, image *storage.Image) {
	// If the image has a nil scan, there is nothing to fill in.
	if image.GetScan() == nil {
		return
	}

	for _, cp := range parts.Children {
		if cp.ComponentV2 == nil {
			log.Errorf("UNEXPECTED: nil component when retrieving components for image %q", image.GetId())
			continue
		}
		// Generate an embedded component from the non-embedded version.
		image.GetScan().Components = append(image.GetScan().Components, generateEmbeddedComponentV2(cp))
	}
}

func generateEmbeddedComponentV2(cp ComponentParts) *storage.EmbeddedImageScanComponent {
	vulns := make([]*storage.EmbeddedVulnerability, 0, len(cp.Children))
	for _, cve := range cp.Children {
		if cve.CVEV2 == nil {
			log.Errorf("UNEXPECTED: nil cve when retrieving cves for component %q", cp.Component.GetId())
			continue
		}
		vulns = append(vulns, utils.ImageCVEV2ToEmbeddedVulnerability(cve.CVEV2))
	}

	ret := &storage.EmbeddedImageScanComponent{
		Name:         cp.ComponentV2.GetName(),
		Version:      cp.ComponentV2.GetVersion(),
		Architecture: cp.ComponentV2.GetArchitecture(),
		Source:       cp.ComponentV2.GetSource(),
		Location:     cp.ComponentV2.GetLocation(),
		FixedBy:      cp.ComponentV2.GetFixedBy(),
		RiskScore:    cp.ComponentV2.GetRiskScore(),
		Priority:     cp.ComponentV2.GetPriority(),
		Vulns:        vulns,
	}

	if cp.ComponentV2.HasLayerIndex != nil {
		ret.HasLayerIndex = &storage.EmbeddedImageScanComponent_LayerIndex{
			LayerIndex: cp.ComponentV2.GetLayerIndex(),
		}
	}

	if cp.ComponentV2.GetSetTopCvss() != nil {
		ret.SetTopCvss = &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: cp.ComponentV2.GetTopCvss()}
	}

	return ret
}
