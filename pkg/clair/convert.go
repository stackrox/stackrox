package clair

import (
	"encoding/json"

	clairV1 "github.com/coreos/clair/api/v1"
	"github.com/stackrox/rox/generated/storage"
	cvssconv "github.com/stackrox/rox/pkg/cvss"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/scans"
	"github.com/stackrox/rox/pkg/stringutils"
)

var (
	log = logging.LoggerForModule()
)

type nvd struct {
	Cvss cvss `json:"CVSSv2"`
}

type cvss struct {
	Score   float32 `json:"score"`
	Vectors string  `json:"vectors"`
}

// ConvertVulnerability converts a clair vulnerability to a proto vulnerability
func ConvertVulnerability(v clairV1.Vulnerability) *storage.Vulnerability {
	if v.Link == "" {
		v.Link = scans.GetVulnLink(v.Name)
	}
	vul := &storage.Vulnerability{
		Cve:     v.Name,
		Summary: stringutils.Truncate(v.Description, 64, stringutils.WordOriented{}),
		Link:    v.Link,
		SetFixedBy: &storage.Vulnerability_FixedBy{
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

func convertFeature(feature clairV1.Feature) *storage.ImageScanComponent {
	component := &storage.ImageScanComponent{
		Name:    feature.Name,
		Version: feature.Version,
	}
	component.Vulns = make([]*storage.Vulnerability, 0, len(feature.Vulnerabilities))
	for _, v := range feature.Vulnerabilities {
		component.Vulns = append(component.GetVulns(), ConvertVulnerability(v))
	}
	return component
}

func buildSHAToIndexMap(image *storage.Image) map[string]int32 {
	layerSHAToIndex := make(map[string]int32)

	if image.GetMetadata().GetV2() != nil {
		var layerIdx int
		for i, l := range image.GetMetadata().GetV1().GetLayers() {
			if !l.Empty {
				if layerIdx >= len(image.Metadata.LayerShas) {
					log.Errorf("More layers than expected when correlating V2 instructions to V1 layers")
					break
				}
				sha := image.GetMetadata().LayerShas[layerIdx]
				layerSHAToIndex[sha] = int32(i)
				layerIdx++
			}
		}
	} else {
		// If it's V1 then we should have a 1:1 mapping of layer SHAs to the layerOrdering slice
		for i := range image.GetMetadata().GetV1().GetLayers() {
			if i >= len(image.Metadata.LayerShas) {
				log.Errorf("More layers than expected when correlating V1 instructions to V1 layers")
				break
			}
			layerSHAToIndex[image.Metadata.LayerShas[i]] = int32(i)
		}
	}
	return layerSHAToIndex
}

// ConvertFeatures converts clair features to proto components
func ConvertFeatures(image *storage.Image, features []clairV1.Feature) (components []*storage.ImageScanComponent) {
	layerSHAToIndex := buildSHAToIndexMap(image)

	components = make([]*storage.ImageScanComponent, 0, len(features))
	for _, feature := range features {
		convertedComponent := convertFeature(feature)
		if val, ok := layerSHAToIndex[feature.AddedBy]; ok {
			convertedComponent.HasLayerIndex = &storage.ImageScanComponent_LayerIndex{
				LayerIndex: val,
			}
		}
		components = append(components, convertedComponent)
	}
	return
}
