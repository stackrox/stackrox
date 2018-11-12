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

// ConvertFeatures converts clair features to proto components
func ConvertFeatures(features []clairV1.Feature) (components []*v1.ImageScanComponent) {
	components = make([]*v1.ImageScanComponent, 0, len(features))
	for _, feature := range features {
		component := &v1.ImageScanComponent{
			Name:    feature.Name,
			Version: feature.Version,
		}
		component.Vulns = make([]*v1.Vulnerability, 0, len(feature.Vulnerabilities))
		for _, v := range feature.Vulnerabilities {
			component.Vulns = append(component.GetVulns(), ConvertVulnerability(v))
		}
		components = append(components, component)
	}
	return
}
