package utils

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/scancomponent"
)

type componentPieces struct {
	imageID     string
	componentID string
	cveIndex    int
}

var (
	ts = protocompat.TimestampNow()

	testComponent = &storage.EmbeddedImageScanComponent{
		Name:         "comp1",
		Version:      "ver2",
		Source:       0,
		Location:     "/test",
		Architecture: "arm",
	}

	testVulns = []*storage.EmbeddedVulnerability{
		{
			Cve:          "cve1",
			Cvss:         0,
			Summary:      "",
			Link:         "",
			SetFixedBy:   nil,
			ScoreVersion: 1,
			CvssV2:       nil,
			CvssV3: &storage.CVSSV3{
				Vector:              "testVector",
				ExploitabilityScore: 1.0,
				ImpactScore:         2.0,
				AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
				AttackComplexity:    storage.CVSSV3_COMPLEXITY_HIGH,
				PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_HIGH,
				UserInteraction:     storage.CVSSV3_UI_REQUIRED,
				Scope:               storage.CVSSV3_CHANGED,
				Confidentiality:     storage.CVSSV3_IMPACT_HIGH,
				Integrity:           storage.CVSSV3_IMPACT_HIGH,
				Availability:        storage.CVSSV3_IMPACT_HIGH,
				Score:               11.0,
				Severity:            storage.CVSSV3_CRITICAL,
			},
			PublishedOn:           ts,
			LastModified:          ts,
			VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
			VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
			Suppressed:            false,
			SuppressActivation:    nil,
			SuppressExpiry:        nil,
			FirstSystemOccurrence: ts,
			FirstImageOccurrence:  ts,
			Severity:              0,
			State:                 0,
			CvssMetrics: []*storage.CVSSScore{
				{
					Source: storage.Source_SOURCE_NVD,
					Url:    "blah.com",
					CvssScore: &storage.CVSSScore_Cvssv3{
						Cvssv3: &storage.CVSSV3{
							Vector:              "testVector",
							ExploitabilityScore: 1.0,
							ImpactScore:         2.0,
							AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
							AttackComplexity:    storage.CVSSV3_COMPLEXITY_HIGH,
							PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_HIGH,
							UserInteraction:     storage.CVSSV3_UI_REQUIRED,
							Scope:               storage.CVSSV3_CHANGED,
							Confidentiality:     storage.CVSSV3_IMPACT_HIGH,
							Integrity:           storage.CVSSV3_IMPACT_HIGH,
							Availability:        storage.CVSSV3_IMPACT_HIGH,
							Score:               11.0,
							Severity:            storage.CVSSV3_CRITICAL,
						},
					},
				},
			},
			NvdCvss: 11,
			Epss: &storage.EPSS{
				EpssProbability: 22,
				EpssPercentile:  98,
			},
		},
		{
			Cve:     "cve2",
			Cvss:    0,
			Summary: "",
			Link:    "",
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
				FixedBy: "ver3",
			},
			ScoreVersion: 0,
			CvssV2: &storage.CVSSV2{
				Vector:              "testVector2",
				AttackVector:        storage.CVSSV2_ATTACK_ADJACENT,
				AccessComplexity:    storage.CVSSV2_ACCESS_LOW,
				Authentication:      storage.CVSSV2_AUTH_NONE,
				Confidentiality:     storage.CVSSV2_IMPACT_COMPLETE,
				Integrity:           storage.CVSSV2_IMPACT_COMPLETE,
				Availability:        storage.CVSSV2_IMPACT_COMPLETE,
				ExploitabilityScore: 22,
				ImpactScore:         32,
				Score:               43,
				Severity:            storage.CVSSV2_HIGH,
			},
			CvssV3:                nil,
			PublishedOn:           ts,
			LastModified:          ts,
			VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
			VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
			Suppressed:            false,
			SuppressActivation:    nil,
			SuppressExpiry:        nil,
			FirstSystemOccurrence: ts,
			FirstImageOccurrence:  ts,
			Severity:              0,
			State:                 0,
			CvssMetrics:           nil,
			NvdCvss:               0,
			Epss:                  nil,
		},
	}

	testCVEs = []*storage.ImageCVEV2{
		{
			Id:      cve.IDV2("cve1", getTestComponentID(), "0"),
			ImageId: "sha",
			CveBaseInfo: &storage.CVEInfo{
				Cve:          "cve1",
				Summary:      "",
				Link:         "",
				PublishedOn:  ts,
				CreatedAt:    ts,
				LastModified: ts,
				ScoreVersion: 1,
				CvssV2:       nil,
				CvssV3: &storage.CVSSV3{
					Vector:              "testVector",
					ExploitabilityScore: 1.0,
					ImpactScore:         2.0,
					AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
					AttackComplexity:    storage.CVSSV3_COMPLEXITY_HIGH,
					PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_HIGH,
					UserInteraction:     storage.CVSSV3_UI_REQUIRED,
					Scope:               storage.CVSSV3_CHANGED,
					Confidentiality:     storage.CVSSV3_IMPACT_HIGH,
					Integrity:           storage.CVSSV3_IMPACT_HIGH,
					Availability:        storage.CVSSV3_IMPACT_HIGH,
					Score:               11.0,
					Severity:            storage.CVSSV3_CRITICAL,
				},
				References: nil,
				CvssMetrics: []*storage.CVSSScore{
					{
						Source: storage.Source_SOURCE_NVD,
						Url:    "blah.com",
						CvssScore: &storage.CVSSScore_Cvssv3{
							Cvssv3: &storage.CVSSV3{
								Vector:              "testVector",
								ExploitabilityScore: 1.0,
								ImpactScore:         2.0,
								AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
								AttackComplexity:    storage.CVSSV3_COMPLEXITY_HIGH,
								PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_HIGH,
								UserInteraction:     storage.CVSSV3_UI_REQUIRED,
								Scope:               storage.CVSSV3_CHANGED,
								Confidentiality:     storage.CVSSV3_IMPACT_HIGH,
								Integrity:           storage.CVSSV3_IMPACT_HIGH,
								Availability:        storage.CVSSV3_IMPACT_HIGH,
								Score:               11.0,
								Severity:            storage.CVSSV3_CRITICAL,
							},
						},
					},
				},
				Epss: &storage.EPSS{
					EpssProbability: 22,
					EpssPercentile:  98,
				},
			},
			Cvss:                 0,
			Severity:             0,
			ImpactScore:          2.0,
			Nvdcvss:              11,
			NvdScoreVersion:      storage.CvssScoreVersion_V3,
			FirstImageOccurrence: ts,
			State:                0,
			IsFixable:            false,
			HasFixedBy:           nil,
			ComponentId:          getTestComponentID(),
		},
		{
			Id:      cve.IDV2("cve2", getTestComponentID(), "1"),
			ImageId: "sha",
			CveBaseInfo: &storage.CVEInfo{
				Cve:          "cve2",
				Summary:      "",
				Link:         "",
				PublishedOn:  ts,
				CreatedAt:    ts,
				LastModified: ts,
				ScoreVersion: 0,
				CvssV2: &storage.CVSSV2{
					Vector:              "testVector2",
					AttackVector:        storage.CVSSV2_ATTACK_ADJACENT,
					AccessComplexity:    storage.CVSSV2_ACCESS_LOW,
					Authentication:      storage.CVSSV2_AUTH_NONE,
					Confidentiality:     storage.CVSSV2_IMPACT_COMPLETE,
					Integrity:           storage.CVSSV2_IMPACT_COMPLETE,
					Availability:        storage.CVSSV2_IMPACT_COMPLETE,
					ExploitabilityScore: 22,
					ImpactScore:         32,
					Score:               43,
					Severity:            storage.CVSSV2_HIGH,
				},
				CvssV3:      nil,
				References:  nil,
				CvssMetrics: nil,
				Epss:        nil,
			},
			Cvss:                 0,
			Severity:             0,
			ImpactScore:          32,
			Nvdcvss:              0,
			NvdScoreVersion:      storage.CvssScoreVersion_UNKNOWN_VERSION,
			FirstImageOccurrence: ts,
			State:                0,
			IsFixable:            true,
			HasFixedBy: &storage.ImageCVEV2_FixedBy{
				FixedBy: "ver3",
			},
			ComponentId: getTestComponentID(),
		},
	}

	componentInfo = []*componentPieces{
		{
			imageID:     "sha",
			componentID: getTestComponentID(),
			cveIndex:    0,
		},
		{
			imageID:     "sha",
			componentID: getTestComponentID(),
			cveIndex:    1,
		},
	}
)

func TestImageCVEV2ToEmbeddedCVEs(t *testing.T) {
	for idx, imageVuln := range testCVEs {
		embeddedVuln := ImageCVEV2ToEmbeddedVulnerability(imageVuln)
		protoassert.Equal(t, testVulns[idx], embeddedVuln)
	}
}

func TestEmbeddedCVEToImageCVEV2(t *testing.T) {
	for idx, embeddedVuln := range testVulns {
		convertedVuln := EmbeddedVulnerabilityToImageCVEV2(componentInfo[idx].imageID, componentInfo[idx].componentID, componentInfo[idx].cveIndex, embeddedVuln)
		protoassert.Equal(t, testCVEs[idx], convertedVuln)
	}
}

func getTestComponentID() string {
	id, _ := scancomponent.ComponentIDV2(testComponent, "sha")

	return id
}
