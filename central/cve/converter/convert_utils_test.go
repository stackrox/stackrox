package converter

import (
	"testing"
	"time"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stretchr/testify/assert"
)

const (
	publishedDateTime    = "2006-01-02T15:04Z"
	lastModifiedDateTime = "2006-05-02T15:04Z"
)

func TestNvdCVEsToProtoCVEs(t *testing.T) {
	cves := []*schema.NVDCVEFeedJSON10DefCVEItem{
		{
			CVE: &schema.CVEJSON40{
				CVEDataMeta: &schema.CVEJSON40CVEDataMeta{
					ID: "cve-2019-1",
				},
				References: &schema.CVEJSON40References{
					ReferenceData: []*schema.CVEJSON40Reference{
						{
							URL: "Reference1",
						},
						{
							URL: "Reference2",
						},
					},
				},
				Description: &schema.CVEJSON40Description{
					DescriptionData: []*schema.CVEJSON40LangString{
						{
							Lang:  "en",
							Value: "Description1",
						},
						{
							Lang:  "en",
							Value: "Description2",
						},
					},
				},
			},
			LastModifiedDate: lastModifiedDateTime,
			PublishedDate:    publishedDateTime,
			Impact: &schema.NVDCVEFeedJSON10DefImpact{
				BaseMetricV2: &schema.NVDCVEFeedJSON10DefImpactBaseMetricV2{
					CVSSV2: &schema.CVSSV20{
						VectorString:          "AV:N/AC:L/Au:N/C:C/I:C/A:C",
						BaseScore:             10,
						AccessVector:          "NETWORK",
						AccessComplexity:      "LOW",
						Authentication:        "NONE",
						ConfidentialityImpact: "COMPLETE",
						IntegrityImpact:       "COMPLETE",
						AvailabilityImpact:    "COMPLETE",
					},
					Severity:            "HIGH",
					ExploitabilityScore: 10,
					ImpactScore:         10,
				},
				BaseMetricV3: &schema.NVDCVEFeedJSON10DefImpactBaseMetricV3{
					CVSSV3: &schema.CVSSV30{
						VectorString:          "CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
						BaseScore:             9.8,
						AttackVector:          "NETWORK",
						AttackComplexity:      "LOW",
						PrivilegesRequired:    "NONE",
						UserInteraction:       "NONE",
						Scope:                 "UNCHANGED",
						ConfidentialityImpact: "HIGH",
						IntegrityImpact:       "HIGH",
						AvailabilityImpact:    "HIGH",
						BaseSeverity:          "CRITICAL",
					},
					ExploitabilityScore: 3.9,
					ImpactScore:         5.9,
				},
			},
		},
		{
			CVE: &schema.CVEJSON40{
				CVEDataMeta: &schema.CVEJSON40CVEDataMeta{
					ID: "cve-2019-2",
				},
				References: &schema.CVEJSON40References{
					ReferenceData: []*schema.CVEJSON40Reference{
						{
							URL: "Reference3",
						},
						{
							URL: "Reference4",
						},
					},
				},
				Description: &schema.CVEJSON40Description{
					DescriptionData: []*schema.CVEJSON40LangString{
						{
							Lang:  "en",
							Value: "Description3",
						},
						{
							Lang:  "en",
							Value: "Description4",
						},
					},
				},
			},
			LastModifiedDate: lastModifiedDateTime,
			PublishedDate:    publishedDateTime,
			Impact: &schema.NVDCVEFeedJSON10DefImpact{
				BaseMetricV2: &schema.NVDCVEFeedJSON10DefImpactBaseMetricV2{
					CVSSV2: &schema.CVSSV20{
						VectorString:          "AV:N/AC:L/Au:S/C:N/I:P/A:N",
						BaseScore:             4,
						AccessVector:          "NETWORK",
						AccessComplexity:      "LOW",
						Authentication:        "SINGLE",
						ConfidentialityImpact: "NONE",
						IntegrityImpact:       "PARTIAL",
						AvailabilityImpact:    "NONE",
					},
					Severity:            "MEDIUM",
					ExploitabilityScore: 8,
					ImpactScore:         2.9,
				},
				BaseMetricV3: &schema.NVDCVEFeedJSON10DefImpactBaseMetricV3{
					CVSSV3: &schema.CVSSV30{
						VectorString:          "CVSS:3.0/AV:N/AC:L/PR:L/UI:N/S:C/C:N/I:H/A:N",
						BaseScore:             7.7,
						AttackVector:          "NETWORK",
						AttackComplexity:      "LOW",
						PrivilegesRequired:    "LOW",
						UserInteraction:       "NONE",
						Scope:                 "CHANGED",
						ConfidentialityImpact: "NONE",
						IntegrityImpact:       "HIGH",
						AvailabilityImpact:    "NONE",
						BaseSeverity:          "HIGH",
					},
					ExploitabilityScore: 3.1,
					ImpactScore:         4,
				},
			},
		},
	}

	expectedVuls := []storage.CVE{
		{
			Id:           "cve-2019-1",
			Cvss:         float32(cves[0].Impact.BaseMetricV3.CVSSV3.BaseScore),
			Summary:      "Description1",
			Link:         "https://nvd.nist.gov/vuln/detail/cve-2019-1",
			ScoreVersion: storage.CVE_V3,
			CvssV2: &storage.CVSSV2{
				Vector:              "AV:N/AC:L/Au:N/C:C/I:C/A:C",
				AttackVector:        storage.CVSSV2_ATTACK_NETWORK,
				AccessComplexity:    storage.CVSSV2_ACCESS_LOW,
				Authentication:      storage.CVSSV2_AUTH_NONE,
				Confidentiality:     storage.CVSSV2_IMPACT_COMPLETE,
				Integrity:           storage.CVSSV2_IMPACT_COMPLETE,
				Availability:        storage.CVSSV2_IMPACT_COMPLETE,
				ExploitabilityScore: float32(cves[0].Impact.BaseMetricV2.ExploitabilityScore),
				ImpactScore:         float32(cves[0].Impact.BaseMetricV2.ImpactScore),
				Score:               float32(cves[0].Impact.BaseMetricV2.CVSSV2.BaseScore),
				Severity:            storage.CVSSV2_HIGH,
			},
			CvssV3: &storage.CVSSV3{
				Vector:              "CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
				ExploitabilityScore: float32(cves[0].Impact.BaseMetricV3.ExploitabilityScore),
				ImpactScore:         float32(cves[0].Impact.BaseMetricV3.ImpactScore),
				AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
				AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
				PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_NONE,
				UserInteraction:     storage.CVSSV3_UI_NONE,
				Scope:               storage.CVSSV3_UNCHANGED,
				Confidentiality:     storage.CVSSV3_IMPACT_HIGH,
				Integrity:           storage.CVSSV3_IMPACT_HIGH,
				Availability:        storage.CVSSV3_IMPACT_HIGH,
				Score:               float32(cves[0].Impact.BaseMetricV3.CVSSV3.BaseScore),
				Severity:            storage.CVSSV3_CRITICAL,
			},
			Type: storage.CVE_K8S_CVE,
		},
		{
			Id:           "cve-2019-2",
			Cvss:         float32(cves[1].Impact.BaseMetricV3.CVSSV3.BaseScore),
			Summary:      "Description3",
			Link:         "https://nvd.nist.gov/vuln/detail/cve-2019-2",
			ScoreVersion: storage.CVE_V3,
			CvssV2: &storage.CVSSV2{
				Vector:              "AV:N/AC:L/Au:S/C:N/I:P/A:N",
				AttackVector:        storage.CVSSV2_ATTACK_NETWORK,
				AccessComplexity:    storage.CVSSV2_ACCESS_LOW,
				Authentication:      storage.CVSSV2_AUTH_SINGLE,
				Confidentiality:     storage.CVSSV2_IMPACT_NONE,
				Integrity:           storage.CVSSV2_IMPACT_PARTIAL,
				Availability:        storage.CVSSV2_IMPACT_NONE,
				ExploitabilityScore: float32(cves[1].Impact.BaseMetricV2.ExploitabilityScore),
				ImpactScore:         float32(cves[1].Impact.BaseMetricV2.ImpactScore),
				Score:               float32(cves[1].Impact.BaseMetricV2.CVSSV2.BaseScore),
				Severity:            storage.CVSSV2_MEDIUM,
			},
			CvssV3: &storage.CVSSV3{
				Vector:              "CVSS:3.0/AV:N/AC:L/PR:L/UI:N/S:C/C:N/I:H/A:N",
				ExploitabilityScore: float32(cves[1].Impact.BaseMetricV3.ExploitabilityScore),
				ImpactScore:         float32(cves[1].Impact.BaseMetricV3.ImpactScore),
				AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
				AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
				PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_LOW,
				UserInteraction:     storage.CVSSV3_UI_NONE,
				Scope:               storage.CVSSV3_CHANGED,
				Confidentiality:     storage.CVSSV3_IMPACT_NONE,
				Integrity:           storage.CVSSV3_IMPACT_HIGH,
				Availability:        storage.CVSSV3_IMPACT_NONE,
				Score:               float32(cves[1].Impact.BaseMetricV3.CVSSV3.BaseScore),
				Severity:            storage.CVSSV3_HIGH,
			},
			Type: storage.CVE_K8S_CVE,
		},
	}

	for i := 0; i < len(cves); i++ {
		expectedVul := &expectedVuls[i]

		ts, err := time.Parse(timeFormat, publishedDateTime)
		assert.Nil(t, err)
		expectedVul.PublishedOn = protoconv.ConvertTimeToTimestamp(ts)

		ts, err = time.Parse(timeFormat, lastModifiedDateTime)
		assert.Nil(t, err)
		expectedVul.LastModified = protoconv.ConvertTimeToTimestamp(ts)

		actualVul, err := NvdCVEToProtoCVE(cves[i], K8s)
		assert.Nil(t, err)
		assert.Equal(t, actualVul, expectedVul)
	}
}
