package clair

import (
	"encoding/json"

	clairV1 "github.com/coreos/clair/api/v1"
	"github.com/stackrox/rox/generated/storage"
	cvssv2 "github.com/stackrox/rox/pkg/cvss/cvssv2"
	cvssv3 "github.com/stackrox/rox/pkg/cvss/cvssv3"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/scans"
	"github.com/stackrox/rox/pkg/stringutils"
)

var (
	log = logging.LoggerForModule()
)

type nvd struct {
	CvssV2 *cvss `json:"CVSSv2"`
	CvssV3 *cvss `json:"CVSSv3"`
}

type cvss struct {
	Score               float32
	Vectors             string
	ExploitabilityScore float32
	ImpactScore         float32
}

// ConvertVulnerability converts a clair vulnerability to a proto vulnerability
func ConvertVulnerability(v clairV1.Vulnerability) *storage.EmbeddedVulnerability {
	if v.Link == "" {
		v.Link = scans.GetVulnLink(v.Name)
	}
	vul := &storage.EmbeddedVulnerability{
		Cve:     v.Name,
		Summary: stringutils.Truncate(v.Description, 64, stringutils.WordOriented{}),
		Link:    v.Link,
		SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
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

		if n.CvssV3 != nil {
			vul.Cvss = n.CvssV3.Score
			if cvssV3, err := cvssv3.ParseCVSSV3(n.CvssV3.Vectors); err == nil && cvssV3.Vector != "" {
				cvssV3.ExploitabilityScore = n.CvssV3.ExploitabilityScore
				cvssV3.ImpactScore = n.CvssV3.ImpactScore
				vul.Vectors = &storage.EmbeddedVulnerability_CvssV3{
					CvssV3: cvssV3,
				}
			}
			vul.ScoreVersion = storage.EmbeddedVulnerability_V3
		} else if n.CvssV2 != nil {
			vul.Cvss = n.CvssV2.Score
			vul.ScoreVersion = storage.EmbeddedVulnerability_V2
			if cvssV2, err := cvssv2.ParseCVSSV2(n.CvssV2.Vectors); err == nil {
				cvssV2.ExploitabilityScore = n.CvssV2.ExploitabilityScore
				cvssV2.ImpactScore = n.CvssV2.ImpactScore
				vul.Vectors = &storage.EmbeddedVulnerability_CvssV2{
					CvssV2: cvssV2,
				}
			} else {
				log.Error(err)
			}
		}
	}
	return vul
}

func convertFeature(feature clairV1.Feature) *storage.EmbeddedImageScanComponent {
	component := &storage.EmbeddedImageScanComponent{
		Name:    feature.Name,
		Version: feature.Version,
	}
	component.Vulns = make([]*storage.EmbeddedVulnerability, 0, len(feature.Vulnerabilities))
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
					log.Error("More layers than expected when correlating V2 instructions to V1 layers")
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
				log.Error("More layers than expected when correlating V1 instructions to V1 layers")
				break
			}
			layerSHAToIndex[image.Metadata.LayerShas[i]] = int32(i)
		}
	}
	return layerSHAToIndex
}

// ConvertFeatures converts clair features to proto components
func ConvertFeatures(image *storage.Image, features []clairV1.Feature) (components []*storage.EmbeddedImageScanComponent) {
	layerSHAToIndex := buildSHAToIndexMap(image)

	components = make([]*storage.EmbeddedImageScanComponent, 0, len(features))
	for _, feature := range features {
		convertedComponent := convertFeature(feature)
		if val, ok := layerSHAToIndex[feature.AddedBy]; ok {
			convertedComponent.HasLayerIndex = &storage.EmbeddedImageScanComponent_LayerIndex{
				LayerIndex: val,
			}
		}
		components = append(components, convertedComponent)
	}
	return
}
