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
		Id:     imageID,
		Digest: imageSha,
		Name: &storage.ImageName{
			FullName: imageName,
		},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Created: ts,
			},
		},
		ScanStats: &storage.ImageV2_ScanStats{
			ComponentCount:         4,
			CveCount:               7,
			FixableCveCount:        4,
			UnknownCveCount:        7,
			FixableUnknownCveCount: 4,
		},
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
			Id:     imageID,
			Digest: imageSha,
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
			ScanStats: &storage.ImageV2_ScanStats{
				ComponentCount:         4,
				CveCount:               7,
				FixableCveCount:        4,
				UnknownCveCount:        7,
				FixableUnknownCveCount: 4,
			},
		},
		Children: []ComponentPartsV2{
			{
				ComponentV2: &storage.ImageComponentV2{
					Id:        getTestComponentID(testComponents[0], imageID, 0),
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
					Id:        getTestComponentID(testComponents[1], imageID, 1),
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
							Id:        getTestCVEID(testCVEs["cve1comp1"], getTestComponentID(testComponents[1], imageID, 1), 0),
							ImageIdV2: imageID,
							CveBaseInfo: &storage.CVEInfo{
								Cve:       "cve1",
								CreatedAt: ts,
							},
							NvdScoreVersion:      storage.CvssScoreVersion_UNKNOWN_VERSION,
							FirstImageOccurrence: ts,
							ComponentId:          getTestComponentID(testComponents[1], imageID, 1),
						},
					},
					{
						CVEV2: &storage.ImageCVEV2{
							Id:        getTestCVEID(testCVEs["cve2comp1"], getTestComponentID(testComponents[1], imageID, 1), 1),
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
							ComponentId:          getTestComponentID(testComponents[1], imageID, 1),
						},
					},
					{
						CVEV2: &storage.ImageCVEV2{
							Id:        getTestCVEID(testCVEs["cve2comp1"], getTestComponentID(testComponents[1], imageID, 1), 2),
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
							ComponentId:          getTestComponentID(testComponents[1], imageID, 1),
						},
					},
				},
			},
			{
				ComponentV2: &storage.ImageComponentV2{
					Id:        getTestComponentID(testComponents[2], imageID, 2),
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
							Id:        getTestCVEID(testCVEs["cve1comp2"], getTestComponentID(testComponents[2], imageID, 2), 0),
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
							ComponentId:          getTestComponentID(testComponents[2], imageID, 2),
						},
					},
					{
						CVEV2: &storage.ImageCVEV2{
							Id:        getTestCVEID(testCVEs["cve2comp2"], getTestComponentID(testComponents[2], imageID, 2), 1),
							ImageIdV2: imageID,
							CveBaseInfo: &storage.CVEInfo{
								Cve:       "cve2",
								CreatedAt: ts,
							},
							NvdScoreVersion:      storage.CvssScoreVersion_UNKNOWN_VERSION,
							FirstImageOccurrence: ts,
							ComponentId:          getTestComponentID(testComponents[2], imageID, 2),
						},
					},
				},
			},
			{
				ComponentV2: &storage.ImageComponentV2{
					Id:        getTestComponentID(testComponents[2], imageID, 3),
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
							Id:        getTestCVEID(testCVEs["cve1comp2"], getTestComponentID(testComponents[2], imageID, 3), 0),
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
							ComponentId:          getTestComponentID(testComponents[2], imageID, 3),
						},
					},
					{
						CVEV2: &storage.ImageCVEV2{
							Id:        getTestCVEID(testCVEs["cve2comp2"], getTestComponentID(testComponents[2], imageID, 3), 1),
							ImageIdV2: imageID,
							CveBaseInfo: &storage.CVEInfo{
								Cve:       "cve2",
								CreatedAt: ts,
							},
							NvdScoreVersion:      storage.CvssScoreVersion_UNKNOWN_VERSION,
							FirstImageOccurrence: ts,
							ComponentId:          getTestComponentID(testComponents[2], imageID, 3),
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

func getTestComponentID(testComponent *storage.EmbeddedImageScanComponent, imageID string, index int) string {
	return scancomponent.ComponentIDV2(testComponent, imageID, index)
}

func getTestCVEID(testCVE *storage.EmbeddedVulnerability, componentID string, index int) string {
	return cve.IDV2(testCVE, componentID, index)
}

func dedupedImageV2(imageID, imageName, imageSha string) *storage.ImageV2 {
	return &storage.ImageV2{
		Id:     imageID,
		Digest: imageSha,
		Name: &storage.ImageName{
			FullName: imageName,
		},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Created: ts,
			},
		},
		ScanStats: &storage.ImageV2_ScanStats{
			ComponentCount:         4,
			CveCount:               7,
			FixableCveCount:        4,
			UnknownCveCount:        7,
			FixableUnknownCveCount: 4,
		},
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
}
