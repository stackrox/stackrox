package common

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

var (
	ts = protocompat.TimestampNow()

	testComponents = []*storage.EmbeddedImageScanComponent{
		{
			Name:    "comp1",
			Version: "ver1",
			HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{
				LayerIndex: 1,
			},
		},
		{
			Name:    "comp1",
			Version: "ver2",
			HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{
				LayerIndex: 3,
			},
			Vulns: []*storage.EmbeddedVulnerability{
				{
					Cve:                   "cve1",
					VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
					FirstImageOccurrence:  ts,
					FirstSystemOccurrence: ts,
				},
				{
					Cve:                "cve2",
					VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
					SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
						FixedBy: "ver3",
					},
					FirstImageOccurrence:  ts,
					FirstSystemOccurrence: ts,
				},
				// Exact duplicate to make sure we filter that out
				{
					Cve:                "cve2",
					VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
					SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
						FixedBy: "ver3",
					},
					FirstImageOccurrence:  ts,
					FirstSystemOccurrence: ts,
				},
			},
		},
		{
			Name:    "comp2",
			Version: "ver1",
			HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{
				LayerIndex: 2,
			},
			Vulns: []*storage.EmbeddedVulnerability{
				{
					Cve:                "cve1",
					VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
					SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
						FixedBy: "ver2",
					},
					FirstImageOccurrence:  ts,
					FirstSystemOccurrence: ts,
				},
				{
					Cve:                   "cve2",
					VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
					FirstImageOccurrence:  ts,
					FirstSystemOccurrence: ts,
				},
			},
		},
		// Exact duplicate to ensure it is filtered out
		{
			Name:    "comp2",
			Version: "ver1",
			HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{
				LayerIndex: 2,
			},
			Vulns: []*storage.EmbeddedVulnerability{
				{
					Cve:                "cve1",
					VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
					SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
						FixedBy: "ver2",
					},
					FirstImageOccurrence:  ts,
					FirstSystemOccurrence: ts,
				},
				{
					Cve:                   "cve2",
					VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
					FirstImageOccurrence:  ts,
					FirstSystemOccurrence: ts,
				},
			},
		},
	}

	testCVEs = map[string]*storage.EmbeddedVulnerability{
		"cve1comp1": {
			Cve:                   "cve1",
			VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
			VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
			FirstImageOccurrence:  ts,
			FirstSystemOccurrence: ts,
		},
		"cve2comp1": {
			Cve:                "cve2",
			VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
			VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
				FixedBy: "ver3",
			},
			FirstImageOccurrence:  ts,
			FirstSystemOccurrence: ts,
		},
		"cve1comp2": {
			Cve:                "cve1",
			VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
			VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
				FixedBy: "ver2",
			},
			FirstImageOccurrence:  ts,
			FirstSystemOccurrence: ts,
		},
		"cve2comp2": {
			Cve:                   "cve2",
			VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
			VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
			FirstImageOccurrence:  ts,
			FirstSystemOccurrence: ts,
		},
	}
)

func TestSplitAndMergeImageV2(t *testing.T) {
	t.Setenv(features.FlattenImageData.EnvVar(), "true")
	if !features.FlattenImageData.Enabled() {
		t.Skip("ROX_FLATTEN_IMAGE_DATA is not enabled")
	}

	imageName := "name"
	imageSha := "sha256:1234567890"
	imageID := uuid.NewV5FromNonUUIDs(imageName, imageSha).String()

	imageV2 := &storage.ImageV2{
		Id:  imageID,
		Sha: imageSha,
		Name: &storage.ImageName{
			FullName: imageName,
		},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Created: ts,
			},
		},
		ComponentCount:         3,
		CveCount:               4,
		FixableCveCount:        2,
		UnknownCveCount:        4,
		FixableUnknownCveCount: 2,
		Scan: &storage.ImageScan{
			ScanTime: ts,
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "comp1",
					Version: "ver1",
					HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{
						LayerIndex: 1,
					},
					Vulns: []*storage.EmbeddedVulnerability{},
				},
				{
					Name:    "comp1",
					Version: "ver2",
					HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{
						LayerIndex: 3,
					},
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:                   "cve1",
							VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							FirstImageOccurrence:  ts,
							FirstSystemOccurrence: ts,
						},
						{
							Cve:                "cve2",
							VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "ver3",
							},
							FirstImageOccurrence:  ts,
							FirstSystemOccurrence: ts,
						},
						{
							Cve:                "cve2",
							VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "ver3",
							},
							FirstImageOccurrence:  ts,
							FirstSystemOccurrence: ts,
						},
					},
				},
				{
					Name:    "comp2",
					Version: "ver1",
					HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{
						LayerIndex: 2,
					},
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:                "cve1",
							VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "ver2",
							},
							FirstImageOccurrence:  ts,
							FirstSystemOccurrence: ts,
						},
						{
							Cve:                   "cve2",
							VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							FirstImageOccurrence:  ts,
							FirstSystemOccurrence: ts,
						},
					},
				},
				{
					Name:    "comp2",
					Version: "ver1",
					HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{
						LayerIndex: 2,
					},
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:                "cve1",
							VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "ver2",
							},
							FirstImageOccurrence:  ts,
							FirstSystemOccurrence: ts,
						},
						{
							Cve:                   "cve2",
							VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							FirstImageOccurrence:  ts,
							FirstSystemOccurrence: ts,
						},
					},
				},
			},
		},
	}

	splitExpected := ImagePartsV2{
		Image: &storage.ImageV2{
			Id:  imageID,
			Sha: imageSha,
			Name: &storage.ImageName{
				FullName: imageName,
			},
			Metadata: &storage.ImageMetadata{
				V1: &storage.V1Metadata{
					Created: ts,
				},
			},
			Scan: &storage.ImageScan{
				ScanTime: ts,
			},
			ComponentCount:         3,
			CveCount:               4,
			FixableCveCount:        2,
			UnknownCveCount:        4,
			FixableUnknownCveCount: 2,
		},
		Children: []ComponentPartsV2{
			{
				ComponentV2: &storage.ImageComponentV2{
					Id:        getTestComponentID(t, testComponents[0], imageID),
					Name:      "comp1",
					Version:   "ver1",
					ImageIdV2: imageID,
					HasLayerIndex: &storage.ImageComponentV2_LayerIndex{
						LayerIndex: 1,
					},
				},
				Children: []CVEPartsV2{},
			},
			{
				ComponentV2: &storage.ImageComponentV2{
					Id:        getTestComponentID(t, testComponents[1], imageID),
					Name:      "comp1",
					Version:   "ver2",
					ImageIdV2: imageID,
					HasLayerIndex: &storage.ImageComponentV2_LayerIndex{
						LayerIndex: 3,
					},
				},
				Children: []CVEPartsV2{
					{
						CVEV2: &storage.ImageCVEV2{
							Id:        getTestCVEID(t, testCVEs["cve1comp1"], getTestComponentID(t, testComponents[1], imageID)),
							ImageIdV2: imageID,
							CveBaseInfo: &storage.CVEInfo{
								Cve:       "cve1",
								CreatedAt: ts,
							},
							NvdScoreVersion:      storage.CvssScoreVersion_UNKNOWN_VERSION,
							FirstImageOccurrence: ts,
							ComponentId:          getTestComponentID(t, testComponents[1], imageID),
						},
					},
					{
						CVEV2: &storage.ImageCVEV2{
							Id:        getTestCVEID(t, testCVEs["cve2comp1"], getTestComponentID(t, testComponents[1], imageID)),
							ImageIdV2: imageID,
							CveBaseInfo: &storage.CVEInfo{
								Cve:       "cve2",
								CreatedAt: ts,
							},
							NvdScoreVersion: storage.CvssScoreVersion_UNKNOWN_VERSION,
							HasFixedBy: &storage.ImageCVEV2_FixedBy{
								FixedBy: "ver3",
							},
							IsFixable:            true,
							FirstImageOccurrence: ts,
							ComponentId:          getTestComponentID(t, testComponents[1], imageID),
						},
					},
				},
			},
			{
				ComponentV2: &storage.ImageComponentV2{
					Id:        getTestComponentID(t, testComponents[2], imageID),
					Name:      "comp2",
					Version:   "ver1",
					ImageIdV2: imageID,
					HasLayerIndex: &storage.ImageComponentV2_LayerIndex{
						LayerIndex: 2,
					},
				},
				Children: []CVEPartsV2{
					{
						CVEV2: &storage.ImageCVEV2{
							Id:        getTestCVEID(t, testCVEs["cve1comp2"], getTestComponentID(t, testComponents[2], imageID)),
							ImageIdV2: imageID,
							CveBaseInfo: &storage.CVEInfo{
								Cve:       "cve1",
								CreatedAt: ts,
							},
							NvdScoreVersion: storage.CvssScoreVersion_UNKNOWN_VERSION,
							HasFixedBy: &storage.ImageCVEV2_FixedBy{
								FixedBy: "ver2",
							},
							IsFixable:            true,
							FirstImageOccurrence: ts,
							ComponentId:          getTestComponentID(t, testComponents[2], imageID),
						},
					},
					{
						CVEV2: &storage.ImageCVEV2{
							Id:        getTestCVEID(t, testCVEs["cve2comp2"], getTestComponentID(t, testComponents[2], imageID)),
							ImageIdV2: imageID,
							CveBaseInfo: &storage.CVEInfo{
								Cve:       "cve2",
								CreatedAt: ts,
							},
							NvdScoreVersion:      storage.CvssScoreVersion_UNKNOWN_VERSION,
							FirstImageOccurrence: ts,
							ComponentId:          getTestComponentID(t, testComponents[2], imageID),
						},
					},
				},
			},
		},
	}

	splitActual, err := Split(imageV2, true)
	assert.NoError(t, err)
	protoassert.Equal(t, splitExpected.Image, splitActual.Image)

	assert.Len(t, splitActual.Children, len(splitExpected.Children))
	for i, expected := range splitExpected.Children {
		actual := splitActual.Children[i]
		protoassert.Equal(t, expected.ComponentV2, actual.ComponentV2)

		assert.Len(t, actual.Children, len(expected.Children), "Expected %d CVEPartsV2, got %d", len(expected.Children), len(actual.Children))
		for i, e := range expected.Children {
			a := actual.Children[i]
			protoassert.Equal(t, e.CVEV2, a.CVEV2)
		}
	}

	imageActual := Merge(splitActual)
	protoassert.Equal(t, dedupedImageV2(imageID, imageName, imageSha), imageActual)
}

func getTestComponentID(t *testing.T, testComponent *storage.EmbeddedImageScanComponent, imageID string) string {
	id, err := scancomponent.ComponentIDV2(testComponent, imageID)
	assert.NoError(t, err)
	return id
}

func getTestCVEID(t *testing.T, testCVE *storage.EmbeddedVulnerability, componentID string) string {
	id, err := cve.IDV2(testCVE, componentID)
	assert.NoError(t, err)
	return id
}

func dedupedImageV2(imageID, imageName, imageSha string) *storage.ImageV2 {
	return &storage.ImageV2{
		Id:  imageID,
		Sha: imageSha,
		Name: &storage.ImageName{
			FullName: imageName,
		},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Created: ts,
			},
		},
		ComponentCount:         3,
		CveCount:               4,
		FixableCveCount:        2,
		UnknownCveCount:        4,
		FixableUnknownCveCount: 2,
		Scan: &storage.ImageScan{
			ScanTime: ts,
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "comp1",
					Version: "ver1",
					HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{
						LayerIndex: 1,
					},
					Vulns: []*storage.EmbeddedVulnerability{},
				},
				{
					Name:    "comp1",
					Version: "ver2",
					HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{
						LayerIndex: 3,
					},
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:                   "cve1",
							VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							FirstImageOccurrence:  ts,
							FirstSystemOccurrence: ts,
						},
						{
							Cve:                "cve2",
							VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "ver3",
							},
							FirstImageOccurrence:  ts,
							FirstSystemOccurrence: ts,
						},
					},
				},
				{
					Name:    "comp2",
					Version: "ver1",
					HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{
						LayerIndex: 2,
					},
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:                "cve1",
							VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "ver2",
							},
							FirstImageOccurrence:  ts,
							FirstSystemOccurrence: ts,
						},
						{
							Cve:                   "cve2",
							VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							FirstImageOccurrence:  ts,
							FirstSystemOccurrence: ts,
						},
					},
				},
			},
		},
	}
}
