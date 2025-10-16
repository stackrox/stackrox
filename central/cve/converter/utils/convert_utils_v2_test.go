package utils

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
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
		storage.EmbeddedVulnerability_builder{
			Cve:          "cve1",
			Cvss:         0,
			Summary:      "",
			Link:         "",
			SetFixedBy:   nil,
			ScoreVersion: 1,
			CvssV2:       nil,
			CvssV3: storage.CVSSV3_builder{
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
			}.Build(),
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
				storage.CVSSScore_builder{
					Source: storage.Source_SOURCE_NVD,
					Url:    "blah.com",
					Cvssv3: storage.CVSSV3_builder{
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
					}.Build(),
				}.Build(),
			},
			NvdCvss: 11,
			Epss: storage.EPSS_builder{
				EpssProbability: 22,
				EpssPercentile:  98,
			}.Build(),
		}.Build(),
		storage.EmbeddedVulnerability_builder{
			Cve:          "cve2",
			Cvss:         0,
			Summary:      "",
			Link:         "",
			FixedBy:      proto.String("ver3"),
			ScoreVersion: 0,
			CvssV2: storage.CVSSV2_builder{
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
			}.Build(),
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
		}.Build(),
	}
)

func TestImageCVEV2ToEmbeddedCVEs(t *testing.T) {
	for idx, imageVuln := range getTestCVEs(t) {
		embeddedVuln := ImageCVEV2ToEmbeddedVulnerability(imageVuln)
		protoassert.Equal(t, testVulns[idx], embeddedVuln)
	}
}

func TestEmbeddedCVEToImageCVEV2(t *testing.T) {
	for idx, embeddedVuln := range testVulns {
		componentInfo := getComponentInfo(t)
		convertedVuln, err := EmbeddedVulnerabilityToImageCVEV2(componentInfo[idx].imageID, componentInfo[idx].componentID, embeddedVuln)
		assert.NoError(t, err)
		protoassert.Equal(t, getTestCVEs(t)[idx], convertedVuln)
	}
}

func getTestComponentID(t *testing.T) string {
	id, err := scancomponent.ComponentIDV2(testComponent, "sha")
	require.NoError(t, err)
	return id
}

func getTestCVEID(t *testing.T, testCVE *storage.EmbeddedVulnerability, componentID string) string {
	id, err := cve.IDV2(testCVE, componentID)
	require.NoError(t, err)
	return id
}

func getTestCVEs(t *testing.T) []*storage.ImageCVEV2 {
	return []*storage.ImageCVEV2{
		storage.ImageCVEV2_builder{
			Id:      getTestCVEID(t, testVulns[0], getTestComponentID(t)),
			ImageId: "sha",
			CveBaseInfo: storage.CVEInfo_builder{
				Cve:          "cve1",
				Summary:      "",
				Link:         "",
				PublishedOn:  ts,
				CreatedAt:    ts,
				LastModified: ts,
				ScoreVersion: 1,
				CvssV2:       nil,
				CvssV3: storage.CVSSV3_builder{
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
				}.Build(),
				References: nil,
				CvssMetrics: []*storage.CVSSScore{
					storage.CVSSScore_builder{
						Source: storage.Source_SOURCE_NVD,
						Url:    "blah.com",
						Cvssv3: storage.CVSSV3_builder{
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
						}.Build(),
					}.Build(),
				},
				Epss: storage.EPSS_builder{
					EpssProbability: 22,
					EpssPercentile:  98,
				}.Build(),
			}.Build(),
			Cvss:                 0,
			Severity:             0,
			ImpactScore:          2.0,
			Nvdcvss:              11,
			NvdScoreVersion:      storage.CvssScoreVersion_V3,
			FirstImageOccurrence: ts,
			State:                0,
			IsFixable:            false,
			HasFixedBy:           nil,
			ComponentId:          getTestComponentID(t),
		}.Build(),
		storage.ImageCVEV2_builder{
			Id:      getTestCVEID(t, testVulns[1], getTestComponentID(t)),
			ImageId: "sha",
			CveBaseInfo: storage.CVEInfo_builder{
				Cve:          "cve2",
				Summary:      "",
				Link:         "",
				PublishedOn:  ts,
				CreatedAt:    ts,
				LastModified: ts,
				ScoreVersion: 0,
				CvssV2: storage.CVSSV2_builder{
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
				}.Build(),
				CvssV3:      nil,
				References:  nil,
				CvssMetrics: nil,
				Epss:        nil,
			}.Build(),
			Cvss:                 0,
			Severity:             0,
			ImpactScore:          32,
			Nvdcvss:              0,
			NvdScoreVersion:      storage.CvssScoreVersion_UNKNOWN_VERSION,
			FirstImageOccurrence: ts,
			State:                0,
			IsFixable:            true,
			FixedBy:              proto.String("ver3"),
			ComponentId:          getTestComponentID(t),
		}.Build(),
	}
}

func getComponentInfo(t *testing.T) []*componentPieces {
	return []*componentPieces{
		{
			imageID:     "sha",
			componentID: getTestComponentID(t),
			cveIndex:    0,
		},
		{
			imageID:     "sha",
			componentID: getTestComponentID(t),
			cveIndex:    1,
		},
	}
}
