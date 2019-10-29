package clair

import (
	"encoding/json"
	"time"

	clairV1 "github.com/coreos/clair/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cvss/cvssv2"
	"github.com/stackrox/rox/pkg/cvss/cvssv3"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/scans"
	"github.com/stackrox/rox/pkg/stringutils"
)

const (
	timeFormat = "2006-01-02T15:04Z"
)

var (
	log = logging.LoggerForModule()
)

type nvd struct {
	PublishedOn  string `json:"PublishedDateTime"`
	LastModified string `json:"LastModifiedDateTime"`
	CvssV2       *cvss  `json:"CVSSv2"`
	CvssV3       *cvss  `json:"CVSSv3"`
}

type cvss struct {
	Score               float32
	Vectors             string
	ExploitabilityScore float32
	ImpactScore         float32
}

// ConvertVulnerability converts a clair vulnerability to a proto vulnerability
func ConvertVulnerability(v clairV1.Vulnerability) *storage.EmbeddedVulnerability {
	if _, ok := v.Metadata["NVD"]; !ok {
		return nil
	}

	if v.Link == "" {
		v.Link = scans.GetVulnLink(v.Name)
	}
	vul := &storage.EmbeddedVulnerability{
		Cve:     v.Name,
		Summary: stringutils.TruncateIf(v.Description, 64, !features.VulnMgmtUI.Enabled(), stringutils.WordOriented{}),
		Link:    v.Link,
		SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
			FixedBy: v.FixedBy,
		},
	}
	nvdMap := v.Metadata["NVD"]
	d, err := json.Marshal(nvdMap)
	if err != nil {
		return vul
	}
	var n nvd
	if err := json.Unmarshal(d, &n); err != nil {
		return vul
	}
	if n.PublishedOn != "" {
		if ts, err := time.Parse(timeFormat, n.PublishedOn); err == nil {
			vul.PublishedOn = protoconv.ConvertTimeToTimestamp(ts)
		}
	}
	if n.LastModified != "" {
		if ts, err := time.Parse(timeFormat, n.LastModified); err == nil {
			vul.LastModified = protoconv.ConvertTimeToTimestamp(ts)
		}
	}

	if n.CvssV2 != nil && n.CvssV2.Vectors != "" {
		if cvssV2, err := cvssv2.ParseCVSSV2(n.CvssV2.Vectors); err == nil {
			cvssV2.ExploitabilityScore = n.CvssV2.ExploitabilityScore
			cvssV2.ImpactScore = n.CvssV2.ImpactScore
			cvssV2.Score = n.CvssV2.Score

			vul.CvssV2 = cvssV2
			// This sets the top level score for use in policies. It will be overwritten if v3 exists
			vul.Cvss = n.CvssV2.Score
			vul.ScoreVersion = storage.EmbeddedVulnerability_V2
			vul.GetCvssV2().Severity = cvssV2Severity(vul.GetCvss())
		} else {
			log.Error(err)
		}
	}

	if n.CvssV3 != nil && n.CvssV3.Vectors != "" {
		if cvssV3, err := cvssv3.ParseCVSSV3(n.CvssV3.Vectors); err == nil {
			cvssV3.ExploitabilityScore = n.CvssV3.ExploitabilityScore
			cvssV3.ImpactScore = n.CvssV3.ImpactScore
			cvssV3.Score = n.CvssV3.Score

			vul.CvssV3 = cvssV3
			vul.Cvss = n.CvssV3.Score
			vul.ScoreVersion = storage.EmbeddedVulnerability_V3
			vul.GetCvssV3().Severity = cvssV3Severity(vul.GetCvss())
		} else {
			log.Error(err)
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
		if convertedVuln := ConvertVulnerability(v); convertedVuln != nil {
			component.Vulns = append(component.Vulns, convertedVuln)
		}
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

func cvssV3Severity(score float32) storage.CVSSV3_Severity {
	switch {
	case score == 0.0:
		return storage.CVSSV3_NONE
	case score <= 3.9:
		return storage.CVSSV3_LOW
	case score <= 6.9:
		return storage.CVSSV3_MEDIUM
	case score <= 8.9:
		return storage.CVSSV3_HIGH
	case score <= 10.0:
		return storage.CVSSV3_CRITICAL
	}
	return storage.CVSSV3_UNKNOWN
}

func cvssV2Severity(score float32) storage.CVSSV2_Severity {
	switch {
	case score <= 3.9:
		return storage.CVSSV2_LOW
	case score <= 6.9:
		return storage.CVSSV2_MEDIUM
	case score <= 10.0:
		return storage.CVSSV2_HIGH
	}
	return storage.CVSSV2_UNKNOWN
}
