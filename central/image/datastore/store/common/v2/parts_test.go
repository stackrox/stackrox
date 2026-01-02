package common

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/scancomponent"
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

func TestSplitAndMergeImage(t *testing.T) {
	image := &storage.Image{
		Id: "sha",
		Name: &storage.ImageName{
			FullName: "name",
		}, BaseImageInfo: []*storage.BaseImageInfo{
			{
				BaseImageId:       "some-id",
				BaseImageFullName: "registry.example.com/ns/base:tag",
				BaseImageDigest:   "sha256:...",
			},
			{
				BaseImageId:       "another-id",
				BaseImageFullName: "registry.example.com/ns/other:tag",
				BaseImageDigest:   "sha256:...",
			},
		},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Created: ts,
			},
		},
		SetComponents: &storage.Image_Components{
			Components: 4,
		},
		SetCves: &storage.Image_Cves{
			Cves: 7,
		},
		SetFixable: &storage.Image_FixableCves{
			FixableCves: 4,
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

	splitExpected := ImageParts{
		Image: &storage.Image{
			Id: "sha",
			Name: &storage.ImageName{
				FullName: "name",
			},
			Metadata: &storage.ImageMetadata{
				V1: &storage.V1Metadata{
					Created: ts,
				},
			},
			BaseImageInfo: []*storage.BaseImageInfo{
				{
					BaseImageId:       "some-id",
					BaseImageFullName: "registry.example.com/ns/base:tag",
					BaseImageDigest:   "sha256:...",
				},
				{
					BaseImageId:       "another-id",
					BaseImageFullName: "registry.example.com/ns/other:tag",
					BaseImageDigest:   "sha256:...",
				},
			},
			Scan: &storage.ImageScan{
				ScanTime: ts,
			},
			SetComponents: &storage.Image_Components{
				Components: 4,
			},
			SetCves: &storage.Image_Cves{
				Cves: 7,
			},
			SetFixable: &storage.Image_FixableCves{
				FixableCves: 4,
			},
		},
		Children: []ComponentParts{
			{
				ComponentV2: &storage.ImageComponentV2{
					Id:      getTestComponentID(testComponents[0], "sha", 0),
					Name:    "comp1",
					Version: "ver1",
					ImageId: "sha",
					HasLayerIndex: &storage.ImageComponentV2_LayerIndex{
						LayerIndex: 1,
					},
					FromBaseImage: true,
				},
				Children: []CVEParts{},
			},
			{
				ComponentV2: &storage.ImageComponentV2{
					Id:      getTestComponentID(testComponents[1], "sha", 1),
					Name:    "comp1",
					Version: "ver2",
					ImageId: "sha",
					HasLayerIndex: &storage.ImageComponentV2_LayerIndex{
						LayerIndex: 3,
					},
					FromBaseImage: true,
				},
				Children: []CVEParts{
					{
						CVEV2: &storage.ImageCVEV2{
							Id:      getTestCVEID(testCVEs["cve1comp1"], getTestComponentID(testComponents[1], "sha", 1), 0),
							ImageId: "sha",
							CveBaseInfo: &storage.CVEInfo{
								Cve:       "cve1",
								CreatedAt: ts,
							},
							NvdScoreVersion:      storage.CvssScoreVersion_UNKNOWN_VERSION,
							FirstImageOccurrence: ts,
							ComponentId:          getTestComponentID(testComponents[1], "sha", 1),
						},
					},
					{
						CVEV2: &storage.ImageCVEV2{
							Id:      getTestCVEID(testCVEs["cve2comp1"], getTestComponentID(testComponents[1], "sha", 1), 1),
							ImageId: "sha",
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
							ComponentId:          getTestComponentID(testComponents[1], "sha", 1),
						},
					},
					{
						CVEV2: &storage.ImageCVEV2{
							Id:      getTestCVEID(testCVEs["cve2comp1"], getTestComponentID(testComponents[1], "sha", 1), 2),
							ImageId: "sha",
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
							ComponentId:          getTestComponentID(testComponents[1], "sha", 1),
						},
					},
				},
			},
			{
				ComponentV2: &storage.ImageComponentV2{
					Id:      getTestComponentID(testComponents[2], "sha", 2),
					Name:    "comp2",
					Version: "ver1",
					ImageId: "sha",
					HasLayerIndex: &storage.ImageComponentV2_LayerIndex{
						LayerIndex: 2,
					},
					FromBaseImage: true,
				},
				Children: []CVEParts{
					{
						CVEV2: &storage.ImageCVEV2{
							Id:      getTestCVEID(testCVEs["cve1comp2"], getTestComponentID(testComponents[2], "sha", 2), 0),
							ImageId: "sha",
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
							ComponentId:          getTestComponentID(testComponents[2], "sha", 2),
						},
					},
					{
						CVEV2: &storage.ImageCVEV2{
							Id:      getTestCVEID(testCVEs["cve2comp2"], getTestComponentID(testComponents[2], "sha", 2), 1),
							ImageId: "sha",
							CveBaseInfo: &storage.CVEInfo{
								Cve:       "cve2",
								CreatedAt: ts,
							},
							NvdScoreVersion:      storage.CvssScoreVersion_UNKNOWN_VERSION,
							FirstImageOccurrence: ts,
							ComponentId:          getTestComponentID(testComponents[2], "sha", 2),
						},
					},
				},
			},
			{
				ComponentV2: &storage.ImageComponentV2{
					Id:      getTestComponentID(testComponents[2], "sha", 3),
					Name:    "comp2",
					Version: "ver1",
					ImageId: "sha",
					HasLayerIndex: &storage.ImageComponentV2_LayerIndex{
						LayerIndex: 2,
					},
					FromBaseImage: true,
				},
				Children: []CVEParts{
					{
						CVEV2: &storage.ImageCVEV2{
							Id:      getTestCVEID(testCVEs["cve1comp2"], getTestComponentID(testComponents[2], "sha", 3), 0),
							ImageId: "sha",
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
							ComponentId:          getTestComponentID(testComponents[2], "sha", 3),
						},
					},
					{
						CVEV2: &storage.ImageCVEV2{
							Id:      getTestCVEID(testCVEs["cve2comp2"], getTestComponentID(testComponents[2], "sha", 3), 1),
							ImageId: "sha",
							CveBaseInfo: &storage.CVEInfo{
								Cve:       "cve2",
								CreatedAt: ts,
							},
							NvdScoreVersion:      storage.CvssScoreVersion_UNKNOWN_VERSION,
							FirstImageOccurrence: ts,
							ComponentId:          getTestComponentID(testComponents[2], "sha", 3),
						},
					},
				},
			},
		},
	}

	splitActual, err := SplitV2(image, true)
	assert.NoError(t, err)
	protoassert.Equal(t, splitExpected.Image, splitActual.Image)

	assert.Len(t, splitActual.Children, len(splitExpected.Children))
	for i, expected := range splitExpected.Children {
		actual := splitActual.Children[i]
		protoassert.Equal(t, expected.ComponentV2, actual.ComponentV2)

		assert.Len(t, actual.Children, len(expected.Children))
		for i, e := range expected.Children {
			a := actual.Children[i]
			protoassert.Equal(t, e.CVEV2, a.CVEV2)
		}
	}

	// Need to add first occurrence edges as otherwise they will be filtered out
	// These values are added on insertion for the DB which is why we will populate them artificially here
	for _, v := range splitActual.ImageCVEEdges {
		v.FirstImageOccurrence = ts
	}

	imageActual := MergeV2(splitActual)
	expectedFinalImage := dedupedImage()
	expectedFinalImage.BaseImageInfo = image.GetBaseImageInfo()
	protoassert.Equal(t, expectedFinalImage, imageActual)
}

func getTestComponentID(testComponent *storage.EmbeddedImageScanComponent, imageID string, index int) string {
	return scancomponent.ComponentIDV2(testComponent, imageID, index)
}

func getTestCVEID(testCVE *storage.EmbeddedVulnerability, componentID string, index int) string {
	return cve.IDV2(testCVE, componentID, index)
}

func dedupedImage() *storage.Image {
	return &storage.Image{
		Id: "sha",
		Name: &storage.ImageName{
			FullName: "name",
		},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Created: ts,
			},
		},
		SetComponents: &storage.Image_Components{
			Components: 4,
		},
		SetCves: &storage.Image_Cves{
			Cves: 7,
		},
		SetFixable: &storage.Image_FixableCves{
			FixableCves: 4,
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
