package types

import (
	"fmt"
	"math"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/facebookincubator/nvdtools/cvss2"
	"github.com/facebookincubator/nvdtools/cvss3"
	"github.com/stackrox/k8s-cves/pkg/validation"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/scanner/database"
)

// NVDTimeLayout is the time layout used by NVD.
const NVDTimeLayout = schema.TimeLayout

// Metadata is the vulnerability metadata.
type Metadata struct {
	PublishedDateTime    string
	LastModifiedDateTime string
	CVSSv2               MetadataCVSSv2
	CVSSv3               MetadataCVSSv3
}

// MetadataCVSSv2 is the CVSSv2 data.
type MetadataCVSSv2 struct {
	Vectors             string
	Score               float64
	ExploitabilityScore float64
	ImpactScore         float64
}

// MetadataCVSSv3 is the CVSSv3.x data.
type MetadataCVSSv3 struct {
	Vectors             string
	Score               float64
	ExploitabilityScore float64
	ImpactScore         float64
}

// GetDatabaseSeverity determines the database.Severity based on the given *Metadata.
// The database.Severity is determined based on the CVSS score(s) and using the proper
// qualitative severity rating scale https://nvd.nist.gov/vuln-metrics/cvss.
func (m *Metadata) GetDatabaseSeverity() database.Severity {
	if m == nil {
		return database.UnknownSeverity
	}
	if m.CVSSv3.Score != 0 {
		score := m.CVSSv3.Score
		switch {
		case score > 0 && score < 4:
			return database.LowSeverity
		case score >= 4 && score < 7:
			return database.MediumSeverity
		case score >= 7 && score < 9:
			return database.HighSeverity
		case score >= 9 && score <= 10:
			return database.CriticalSeverity
		}
	}
	if m.CVSSv2.Score != 0 {
		score := m.CVSSv2.Score
		switch {
		case score > 0 && score < 4:
			return database.LowSeverity
		case score >= 4 && score < 7:
			return database.MediumSeverity
		case score >= 7 && score <= 10:
			return database.HighSeverity
		}
	}
	return database.UnknownSeverity
}

// NewVulnerability creates a new vulnerability based on the given NVD CVE.
func NewVulnerability(cveitem *schema.NVDCVEFeedJSON10DefCVEItem) *database.Vulnerability {
	metadata := ConvertNVDMetadata(cveitem)
	return &database.Vulnerability{
		Name:        cveitem.CVE.CVEDataMeta.ID,
		Description: ConvertNVDSummary(cveitem),
		Link:        fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", cveitem.CVE.CVEDataMeta.ID),
		Metadata: map[string]interface{}{
			"NVD": metadata,
		},
		Severity: metadata.GetDatabaseSeverity(),
	}
}

// ConvertNVDSummary retrieves the given NVD CVE item's description.
func ConvertNVDSummary(item *schema.NVDCVEFeedJSON10DefCVEItem) string {
	if item == nil || item.CVE == nil || item.CVE.Description == nil {
		return ""
	}
	for _, desc := range item.CVE.Description.DescriptionData {
		if desc.Lang == "en" {
			return desc.Value
		}
	}
	return ""
}

// ConvertNVDMetadata converts the given NVD CVE item into *Metadata.
func ConvertNVDMetadata(item *schema.NVDCVEFeedJSON10DefCVEItem) *Metadata {
	if item == nil {
		return nil
	}
	metadata := &Metadata{
		PublishedDateTime:    item.PublishedDate,
		LastModifiedDateTime: item.LastModifiedDate,
	}
	if impact := item.Impact; impact != nil {
		if impact.BaseMetricV2 != nil && impact.BaseMetricV2.CVSSV2 != nil {
			metadata.CVSSv2 = MetadataCVSSv2{
				Vectors:             item.Impact.BaseMetricV2.CVSSV2.VectorString,
				Score:               item.Impact.BaseMetricV2.CVSSV2.BaseScore,
				ExploitabilityScore: item.Impact.BaseMetricV2.ExploitabilityScore,
				ImpactScore:         item.Impact.BaseMetricV2.ImpactScore,
			}
		}
		if impact.BaseMetricV3 != nil && impact.BaseMetricV3.CVSSV3 != nil {
			metadata.CVSSv3 = MetadataCVSSv3{
				Vectors:             item.Impact.BaseMetricV3.CVSSV3.VectorString,
				Score:               item.Impact.BaseMetricV3.CVSSV3.BaseScore,
				ExploitabilityScore: item.Impact.BaseMetricV3.ExploitabilityScore,
				ImpactScore:         item.Impact.BaseMetricV3.ImpactScore,
			}
		}
	}
	return metadata
}

// ConvertMetadataFromK8s takes the Kubernetes' vulnerability definition,
// and it returns *Metadata based on the given data.
func ConvertMetadataFromK8s(cve *validation.CVESchema) (*Metadata, error) {
	var m Metadata
	if nvd := cve.CVSS.NVD; nvd != nil {
		if nvd.VectorV2 != "" && nvd.ScoreV2 > 0 {
			cvssv2, err := ConvertCVSSv2(nvd.VectorV2)
			if err != nil {
				return nil, err
			}
			m.CVSSv2 = *cvssv2
		}
		if nvd.VectorV3 != "" && nvd.ScoreV3 > 0 {
			cvssv3, err := ConvertCVSSv3(nvd.VectorV3)
			if err != nil {
				return nil, err
			}
			m.CVSSv3 = *cvssv3
		}
	}
	if k8s := cve.CVSS.Kubernetes; k8s != nil {
		if k8s.VectorV3 != "" && k8s.ScoreV3 > 0 {
			cvssv3, err := ConvertCVSSv3(k8s.VectorV3)
			if err != nil {
				return nil, err
			}
			m.CVSSv3 = *cvssv3
		}
	}

	m.PublishedDateTime = cve.Published.Format(NVDTimeLayout)

	return &m, nil
}

// ConvertCVSSv2 converts the given CVSS2 vector into MetadataCVSSv2.
func ConvertCVSSv2(cvss2Vector string) (*MetadataCVSSv2, error) {
	v, err := cvss2.VectorFromString(cvss2Vector)
	if err != nil {
		return nil, err
	}
	if err := v.Validate(); err != nil {
		return nil, err
	}
	var m MetadataCVSSv2
	m.Score = v.BaseScore()
	m.Vectors = cvss2Vector
	m.ExploitabilityScore = roundTo1Decimal(v.ExploitabilityScore())
	m.ImpactScore = roundTo1Decimal(v.ImpactScore(false))
	return &m, nil
}

// ConvertCVSSv3 converts the given CVSS3 vector into MetadataCVSSv3.
func ConvertCVSSv3(cvss3Vector string) (*MetadataCVSSv3, error) {
	v, err := cvss3.VectorFromString(cvss3Vector)
	if err != nil {
		return nil, err
	}
	if err := v.Validate(); err != nil {
		return nil, err
	}

	var m MetadataCVSSv3
	m.Score = v.BaseScore()
	m.Vectors = cvss3Vector
	m.ExploitabilityScore = roundTo1Decimal(v.ExploitabilityScore())
	m.ImpactScore = roundTo1Decimal(v.ImpactScore())
	return &m, nil
}

// roundTo1Decimal returns the given float64 rounded to the nearest tenth place.
func roundTo1Decimal(x float64) float64 {
	return math.Round(x*10) / 10
}

// IsNilOrEmpty returns "true" if the passed Metadata is nil or its contents are empty
func (m *Metadata) IsNilOrEmpty() bool {
	if m == nil {
		return true
	}

	return stringutils.AllEmpty(m.LastModifiedDateTime, m.PublishedDateTime, m.CVSSv2.Vectors, m.CVSSv3.Vectors)
}
