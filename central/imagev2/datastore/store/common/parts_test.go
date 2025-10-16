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
	"google.golang.org/protobuf/proto"
)

var (
	ts = protocompat.TimestampNow()

	testComponents = []*storage.EmbeddedImageScanComponent{
		storage.EmbeddedImageScanComponent_builder{
			Name:       "comp1",
			Version:    "ver1",
			LayerIndex: proto.Int32(1),
		}.Build(),
		storage.EmbeddedImageScanComponent_builder{
			Name:       "comp1",
			Version:    "ver2",
			LayerIndex: proto.Int32(3),
			Vulns: []*storage.EmbeddedVulnerability{
				storage.EmbeddedVulnerability_builder{
					Cve:                   "cve1",
					VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
					FirstImageOccurrence:  ts,
					FirstSystemOccurrence: ts,
				}.Build(),
				storage.EmbeddedVulnerability_builder{
					Cve:                   "cve2",
					VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
					FixedBy:               proto.String("ver3"),
					FirstImageOccurrence:  ts,
					FirstSystemOccurrence: ts,
				}.Build(),
				// Exact duplicate to make sure we filter that out
				storage.EmbeddedVulnerability_builder{
					Cve:                   "cve2",
					VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
					FixedBy:               proto.String("ver3"),
					FirstImageOccurrence:  ts,
					FirstSystemOccurrence: ts,
				}.Build(),
			},
		}.Build(),
		storage.EmbeddedImageScanComponent_builder{
			Name:       "comp2",
			Version:    "ver1",
			LayerIndex: proto.Int32(2),
			Vulns: []*storage.EmbeddedVulnerability{
				storage.EmbeddedVulnerability_builder{
					Cve:                   "cve1",
					VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
					FixedBy:               proto.String("ver2"),
					FirstImageOccurrence:  ts,
					FirstSystemOccurrence: ts,
				}.Build(),
				storage.EmbeddedVulnerability_builder{
					Cve:                   "cve2",
					VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
					FirstImageOccurrence:  ts,
					FirstSystemOccurrence: ts,
				}.Build(),
			},
		}.Build(),
		// Exact duplicate to ensure it is filtered out
		storage.EmbeddedImageScanComponent_builder{
			Name:       "comp2",
			Version:    "ver1",
			LayerIndex: proto.Int32(2),
			Vulns: []*storage.EmbeddedVulnerability{
				storage.EmbeddedVulnerability_builder{
					Cve:                   "cve1",
					VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
					FixedBy:               proto.String("ver2"),
					FirstImageOccurrence:  ts,
					FirstSystemOccurrence: ts,
				}.Build(),
				storage.EmbeddedVulnerability_builder{
					Cve:                   "cve2",
					VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
					FirstImageOccurrence:  ts,
					FirstSystemOccurrence: ts,
				}.Build(),
			},
		}.Build(),
	}

	testCVEs = map[string]*storage.EmbeddedVulnerability{
		"cve1comp1": storage.EmbeddedVulnerability_builder{
			Cve:                   "cve1",
			VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
			VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
			FirstImageOccurrence:  ts,
			FirstSystemOccurrence: ts,
		}.Build(),
		"cve2comp1": storage.EmbeddedVulnerability_builder{
			Cve:                   "cve2",
			VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
			VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
			FixedBy:               proto.String("ver3"),
			FirstImageOccurrence:  ts,
			FirstSystemOccurrence: ts,
		}.Build(),
		"cve1comp2": storage.EmbeddedVulnerability_builder{
			Cve:                   "cve1",
			VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
			VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
			FixedBy:               proto.String("ver2"),
			FirstImageOccurrence:  ts,
			FirstSystemOccurrence: ts,
		}.Build(),
		"cve2comp2": storage.EmbeddedVulnerability_builder{
			Cve:                   "cve2",
			VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
			VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
			FirstImageOccurrence:  ts,
			FirstSystemOccurrence: ts,
		}.Build(),
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

	imageV2 := storage.ImageV2_builder{
		Id:     imageID,
		Digest: imageSha,
		Name: storage.ImageName_builder{
			FullName: imageName,
		}.Build(),
		Metadata: storage.ImageMetadata_builder{
			V1: storage.V1Metadata_builder{
				Created: ts,
			}.Build(),
		}.Build(),
		ScanStats: storage.ImageV2_ScanStats_builder{
			ComponentCount:         3,
			CveCount:               4,
			FixableCveCount:        2,
			UnknownCveCount:        4,
			FixableUnknownCveCount: 2,
		}.Build(),
		Scan: storage.ImageScan_builder{
			ScanTime: ts,
			Components: []*storage.EmbeddedImageScanComponent{
				storage.EmbeddedImageScanComponent_builder{
					Name:       "comp1",
					Version:    "ver1",
					LayerIndex: proto.Int32(1),
					Vulns:      []*storage.EmbeddedVulnerability{},
				}.Build(),
				storage.EmbeddedImageScanComponent_builder{
					Name:       "comp1",
					Version:    "ver2",
					LayerIndex: proto.Int32(3),
					Vulns: []*storage.EmbeddedVulnerability{
						storage.EmbeddedVulnerability_builder{
							Cve:                   "cve1",
							VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							FirstImageOccurrence:  ts,
							FirstSystemOccurrence: ts,
						}.Build(),
						storage.EmbeddedVulnerability_builder{
							Cve:                   "cve2",
							VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							FixedBy:               proto.String("ver3"),
							FirstImageOccurrence:  ts,
							FirstSystemOccurrence: ts,
						}.Build(),
						storage.EmbeddedVulnerability_builder{
							Cve:                   "cve2",
							VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							FixedBy:               proto.String("ver3"),
							FirstImageOccurrence:  ts,
							FirstSystemOccurrence: ts,
						}.Build(),
					},
				}.Build(),
				storage.EmbeddedImageScanComponent_builder{
					Name:       "comp2",
					Version:    "ver1",
					LayerIndex: proto.Int32(2),
					Vulns: []*storage.EmbeddedVulnerability{
						storage.EmbeddedVulnerability_builder{
							Cve:                   "cve1",
							VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							FixedBy:               proto.String("ver2"),
							FirstImageOccurrence:  ts,
							FirstSystemOccurrence: ts,
						}.Build(),
						storage.EmbeddedVulnerability_builder{
							Cve:                   "cve2",
							VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							FirstImageOccurrence:  ts,
							FirstSystemOccurrence: ts,
						}.Build(),
					},
				}.Build(),
				storage.EmbeddedImageScanComponent_builder{
					Name:       "comp2",
					Version:    "ver1",
					LayerIndex: proto.Int32(2),
					Vulns: []*storage.EmbeddedVulnerability{
						storage.EmbeddedVulnerability_builder{
							Cve:                   "cve1",
							VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							FixedBy:               proto.String("ver2"),
							FirstImageOccurrence:  ts,
							FirstSystemOccurrence: ts,
						}.Build(),
						storage.EmbeddedVulnerability_builder{
							Cve:                   "cve2",
							VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							FirstImageOccurrence:  ts,
							FirstSystemOccurrence: ts,
						}.Build(),
					},
				}.Build(),
			},
		}.Build(),
	}.Build()

	splitExpected := ImagePartsV2{
		Image: storage.ImageV2_builder{
			Id:     imageID,
			Digest: imageSha,
			Name: storage.ImageName_builder{
				FullName: imageName,
			}.Build(),
			Metadata: storage.ImageMetadata_builder{
				V1: storage.V1Metadata_builder{
					Created: ts,
				}.Build(),
			}.Build(),
			Scan: storage.ImageScan_builder{
				ScanTime: ts,
			}.Build(),
			ScanStats: storage.ImageV2_ScanStats_builder{
				ComponentCount:         3,
				CveCount:               4,
				FixableCveCount:        2,
				UnknownCveCount:        4,
				FixableUnknownCveCount: 2,
			}.Build(),
		}.Build(),
		Children: []ComponentPartsV2{
			{
				ComponentV2: storage.ImageComponentV2_builder{
					Id:         getTestComponentID(t, testComponents[0], imageID),
					Name:       "comp1",
					Version:    "ver1",
					ImageIdV2:  imageID,
					LayerIndex: proto.Int32(1),
				}.Build(),
				Children: []CVEPartsV2{},
			},
			{
				ComponentV2: storage.ImageComponentV2_builder{
					Id:         getTestComponentID(t, testComponents[1], imageID),
					Name:       "comp1",
					Version:    "ver2",
					ImageIdV2:  imageID,
					LayerIndex: proto.Int32(3),
				}.Build(),
				Children: []CVEPartsV2{
					{
						CVEV2: storage.ImageCVEV2_builder{
							Id:        getTestCVEID(t, testCVEs["cve1comp1"], getTestComponentID(t, testComponents[1], imageID)),
							ImageIdV2: imageID,
							CveBaseInfo: storage.CVEInfo_builder{
								Cve:       "cve1",
								CreatedAt: ts,
							}.Build(),
							NvdScoreVersion:      storage.CvssScoreVersion_UNKNOWN_VERSION,
							FirstImageOccurrence: ts,
							ComponentId:          getTestComponentID(t, testComponents[1], imageID),
						}.Build(),
					},
					{
						CVEV2: storage.ImageCVEV2_builder{
							Id:        getTestCVEID(t, testCVEs["cve2comp1"], getTestComponentID(t, testComponents[1], imageID)),
							ImageIdV2: imageID,
							CveBaseInfo: storage.CVEInfo_builder{
								Cve:       "cve2",
								CreatedAt: ts,
							}.Build(),
							NvdScoreVersion:      storage.CvssScoreVersion_UNKNOWN_VERSION,
							FixedBy:              proto.String("ver3"),
							IsFixable:            true,
							FirstImageOccurrence: ts,
							ComponentId:          getTestComponentID(t, testComponents[1], imageID),
						}.Build(),
					},
				},
			},
			{
				ComponentV2: storage.ImageComponentV2_builder{
					Id:         getTestComponentID(t, testComponents[2], imageID),
					Name:       "comp2",
					Version:    "ver1",
					ImageIdV2:  imageID,
					LayerIndex: proto.Int32(2),
				}.Build(),
				Children: []CVEPartsV2{
					{
						CVEV2: storage.ImageCVEV2_builder{
							Id:        getTestCVEID(t, testCVEs["cve1comp2"], getTestComponentID(t, testComponents[2], imageID)),
							ImageIdV2: imageID,
							CveBaseInfo: storage.CVEInfo_builder{
								Cve:       "cve1",
								CreatedAt: ts,
							}.Build(),
							NvdScoreVersion:      storage.CvssScoreVersion_UNKNOWN_VERSION,
							FixedBy:              proto.String("ver2"),
							IsFixable:            true,
							FirstImageOccurrence: ts,
							ComponentId:          getTestComponentID(t, testComponents[2], imageID),
						}.Build(),
					},
					{
						CVEV2: storage.ImageCVEV2_builder{
							Id:        getTestCVEID(t, testCVEs["cve2comp2"], getTestComponentID(t, testComponents[2], imageID)),
							ImageIdV2: imageID,
							CveBaseInfo: storage.CVEInfo_builder{
								Cve:       "cve2",
								CreatedAt: ts,
							}.Build(),
							NvdScoreVersion:      storage.CvssScoreVersion_UNKNOWN_VERSION,
							FirstImageOccurrence: ts,
							ComponentId:          getTestComponentID(t, testComponents[2], imageID),
						}.Build(),
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
	return storage.ImageV2_builder{
		Id:     imageID,
		Digest: imageSha,
		Name: storage.ImageName_builder{
			FullName: imageName,
		}.Build(),
		Metadata: storage.ImageMetadata_builder{
			V1: storage.V1Metadata_builder{
				Created: ts,
			}.Build(),
		}.Build(),
		ScanStats: storage.ImageV2_ScanStats_builder{
			ComponentCount:         3,
			CveCount:               4,
			FixableCveCount:        2,
			UnknownCveCount:        4,
			FixableUnknownCveCount: 2,
		}.Build(),
		Scan: storage.ImageScan_builder{
			ScanTime: ts,
			Components: []*storage.EmbeddedImageScanComponent{
				storage.EmbeddedImageScanComponent_builder{
					Name:       "comp1",
					Version:    "ver1",
					LayerIndex: proto.Int32(1),
					Vulns:      []*storage.EmbeddedVulnerability{},
				}.Build(),
				storage.EmbeddedImageScanComponent_builder{
					Name:       "comp1",
					Version:    "ver2",
					LayerIndex: proto.Int32(3),
					Vulns: []*storage.EmbeddedVulnerability{
						storage.EmbeddedVulnerability_builder{
							Cve:                   "cve1",
							VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							FirstImageOccurrence:  ts,
							FirstSystemOccurrence: ts,
						}.Build(),
						storage.EmbeddedVulnerability_builder{
							Cve:                   "cve2",
							VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							FixedBy:               proto.String("ver3"),
							FirstImageOccurrence:  ts,
							FirstSystemOccurrence: ts,
						}.Build(),
					},
				}.Build(),
				storage.EmbeddedImageScanComponent_builder{
					Name:       "comp2",
					Version:    "ver1",
					LayerIndex: proto.Int32(2),
					Vulns: []*storage.EmbeddedVulnerability{
						storage.EmbeddedVulnerability_builder{
							Cve:                   "cve1",
							VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							FixedBy:               proto.String("ver2"),
							FirstImageOccurrence:  ts,
							FirstSystemOccurrence: ts,
						}.Build(),
						storage.EmbeddedVulnerability_builder{
							Cve:                   "cve2",
							VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							FirstImageOccurrence:  ts,
							FirstSystemOccurrence: ts,
						}.Build(),
					},
				}.Build(),
			},
		}.Build(),
	}.Build()
}
