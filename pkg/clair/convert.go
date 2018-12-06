package clair

import (
	"encoding/json"

	clairV1 "github.com/coreos/clair/api/v1"
	"github.com/stackrox/rox/generated/api/v1"
	cvssconv "github.com/stackrox/rox/pkg/cvss"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/scans"
)

var log = logging.LoggerForModule()

type nvd struct {
	Cvss cvss `json:"CVSSv2"`
}

type cvss struct {
	Score   float32 `json:"score"`
	Vectors string  `json:"vectors"`
}

// ConvertVulnerability converts a clair vulnerability to a proto vulnerability
func ConvertVulnerability(v clairV1.Vulnerability) *v1.Vulnerability {
	if v.Link == "" {
		v.Link = scans.GetVulnLink(v.Name)
	}
	vul := &v1.Vulnerability{
		Cve:     v.Name,
		Summary: v.Description,
		Link:    v.Link,
		SetFixedBy: &v1.Vulnerability_FixedBy{
			FixedBy: v.FixedBy,
		},
	}
	if nvdMap, ok := v.Metadata["NVD"]; ok {
		d, err := json.Marshal(nvdMap)
		if err != nil {
			return vul
		}
		var n nvd
		if err := json.Unmarshal(d, &n); err != nil {
			return vul
		}
		vul.Cvss = n.Cvss.Score
		if cvssVector, err := cvssconv.ParseCVSSV2(n.Cvss.Vectors); err == nil {
			vul.CvssV2 = cvssVector
		}
	}
	return vul
}

// PopulateLayersWithScan derives the layers from the Clair layer envelope
func PopulateLayersWithScan(image *v1.Image, envelope *clairV1.LayerEnvelope) {
	// if the image metadata is empty then simply return
	if len(image.GetMetadata().GetLayerShas()) == 0 || image.GetMetadata().GetV1() == nil {
		return
	}

	// Generate a map of layer shas -> components with CVEs
	// Not all of the layers will be represent (only ones with components and vulnerabilities)
	layers := make(map[string][]*v1.ImageScanComponent)
	for _, f := range envelope.Layer.Features {
		layers[f.AddedBy] = append(layers[f.AddedBy], convertFeature(f))
	}

	if len(layers) > len(image.GetMetadata().GetLayerShas()) {
		log.Warnf("More layers with vulnerabilities than expected: expected %d, but got %d. Scans may be mis-attributed for %s", len(image.GetMetadata().GetLayerShas()), len(layers), image.GetName().GetFullName())
	}

	// Create a slice that is ordered by the layer SHAs so that we can attribute them to the V1 SHAs
	// This will allow us to interpolate the version of the layers
	var layerOrdering [][]*v1.ImageScanComponent
	for _, l := range image.GetMetadata().GetLayerShas() {
		layerOrdering = append(layerOrdering, layers[l])
	}

	// If we have V2, then layer shas is from V2 manifest and this means that there can be fewer layers and than v1
	if image.GetMetadata().GetV2() != nil {
		layerIdx := 0
		for _, l := range image.GetMetadata().GetV1().GetLayers() {
			if !l.Empty {
				// For safety purposes, if layerIdx >= len(layerOrdering) then log a warning
				if layerIdx >= len(layerOrdering) {
					log.Errorf("More layers than expected when correlating V2 instructions to V1 layers")
					break
				}
				l.Components = layerOrdering[layerIdx]
				layerIdx++
			}
		}
	} else {
		// If it's V1 then we should have a 1:1 mapping of layer SHAs to the layerOrdering slice
		for i, l := range image.GetMetadata().GetV1().GetLayers() {
			l.Components = layerOrdering[i]
		}
	}
}

func convertFeature(feature clairV1.Feature) *v1.ImageScanComponent {
	component := &v1.ImageScanComponent{
		Name:    feature.Name,
		Version: feature.Version,
	}
	component.Vulns = make([]*v1.Vulnerability, 0, len(feature.Vulnerabilities))
	for _, v := range feature.Vulnerabilities {
		component.Vulns = append(component.GetVulns(), ConvertVulnerability(v))
	}
	return component
}

// ConvertFeatures converts clair features to proto components
func ConvertFeatures(features []clairV1.Feature) (components []*v1.ImageScanComponent) {
	components = make([]*v1.ImageScanComponent, 0, len(features))
	for _, feature := range features {
		components = append(components, convertFeature(feature))
	}
	return
}
