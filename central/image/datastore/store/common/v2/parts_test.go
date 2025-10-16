package common

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/scancomponent"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
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

func TestSplitAndMergeImage(t *testing.T) {
	image := storage.Image_builder{
		Id: "sha",
		Name: storage.ImageName_builder{
			FullName: "name",
		}.Build(),
		Metadata: storage.ImageMetadata_builder{
			V1: storage.V1Metadata_builder{
				Created: ts,
			}.Build(),
		}.Build(),
		Components:  proto.Int32(3),
		Cves:        proto.Int32(4),
		FixableCves: proto.Int32(2),
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

	splitExpected := ImageParts{
		Image: storage.Image_builder{
			Id: "sha",
			Name: storage.ImageName_builder{
				FullName: "name",
			}.Build(),
			Metadata: storage.ImageMetadata_builder{
				V1: storage.V1Metadata_builder{
					Created: ts,
				}.Build(),
			}.Build(),
			Scan: storage.ImageScan_builder{
				ScanTime: ts,
			}.Build(),
			Components:  proto.Int32(3),
			Cves:        proto.Int32(4),
			FixableCves: proto.Int32(2),
		}.Build(),
		ImageCVEEdges: map[string]*storage.ImageCVEEdge{
			cve.ID("cve1", ""): storage.ImageCVEEdge_builder{
				Id:         pgSearch.IDFromPks([]string{"sha", cve.ID("cve1", "")}),
				ImageId:    "sha",
				ImageCveId: cve.ID("cve1", ""),
			}.Build(),
			cve.ID("cve2", ""): storage.ImageCVEEdge_builder{
				Id:         pgSearch.IDFromPks([]string{"sha", cve.ID("cve2", "")}),
				ImageId:    "sha",
				ImageCveId: cve.ID("cve2", ""),
			}.Build(),
		},
		Children: []ComponentParts{
			{
				Component: storage.ImageComponent_builder{
					Id:      scancomponent.ComponentID("comp1", "ver1", ""),
					Name:    "comp1",
					Version: "ver1",
				}.Build(),
				ComponentV2: storage.ImageComponentV2_builder{
					Id:         getTestComponentID(testComponents[0], "sha"),
					Name:       "comp1",
					Version:    "ver1",
					ImageId:    "sha",
					LayerIndex: proto.Int32(1),
				}.Build(),
				Edge: storage.ImageComponentEdge_builder{
					Id:               pgSearch.IDFromPks([]string{"sha", scancomponent.ComponentID("comp1", "ver1", "")}),
					ImageId:          "sha",
					ImageComponentId: scancomponent.ComponentID("comp1", "ver1", ""),
					LayerIndex:       proto.Int32(1),
				}.Build(),
				Children: []CVEParts{},
			},
			{
				Component: storage.ImageComponent_builder{
					Id:      scancomponent.ComponentID("comp1", "ver2", ""),
					Name:    "comp1",
					Version: "ver2",
				}.Build(),
				ComponentV2: storage.ImageComponentV2_builder{
					Id:         getTestComponentID(testComponents[1], "sha"),
					Name:       "comp1",
					Version:    "ver2",
					ImageId:    "sha",
					LayerIndex: proto.Int32(3),
				}.Build(),
				Edge: storage.ImageComponentEdge_builder{
					Id:               pgSearch.IDFromPks([]string{"sha", scancomponent.ComponentID("comp1", "ver2", "")}),
					ImageId:          "sha",
					ImageComponentId: scancomponent.ComponentID("comp1", "ver2", ""),
					LayerIndex:       proto.Int32(3),
				}.Build(),
				Children: []CVEParts{
					{
						CVE: storage.ImageCVE_builder{
							Id: cve.ID("cve1", ""),
							CveBaseInfo: storage.CVEInfo_builder{
								Cve:       "cve1",
								CreatedAt: ts,
							}.Build(),
							NvdScoreVersion: storage.CvssScoreVersion_UNKNOWN_VERSION,
						}.Build(),
						Edge: storage.ComponentCVEEdge_builder{
							Id:               pgSearch.IDFromPks([]string{scancomponent.ComponentID("comp1", "ver2", ""), cve.ID("cve1", "")}),
							ImageComponentId: scancomponent.ComponentID("comp1", "ver2", ""),
							ImageCveId:       cve.ID("cve1", ""),
						}.Build(),
						CVEV2: storage.ImageCVEV2_builder{
							Id:      getTestCVEID(testCVEs["cve1comp1"], getTestComponentID(testComponents[1], "sha")),
							ImageId: "sha",
							CveBaseInfo: storage.CVEInfo_builder{
								Cve:       "cve1",
								CreatedAt: ts,
							}.Build(),
							NvdScoreVersion:      storage.CvssScoreVersion_UNKNOWN_VERSION,
							FirstImageOccurrence: ts,
							ComponentId:          getTestComponentID(testComponents[1], "sha"),
						}.Build(),
					},
					{
						CVE: storage.ImageCVE_builder{
							Id: cve.ID("cve2", ""),
							CveBaseInfo: storage.CVEInfo_builder{
								Cve:       "cve2",
								CreatedAt: ts,
							}.Build(),
							NvdScoreVersion: storage.CvssScoreVersion_UNKNOWN_VERSION,
						}.Build(),
						Edge: storage.ComponentCVEEdge_builder{
							Id:               pgSearch.IDFromPks([]string{scancomponent.ComponentID("comp1", "ver2", ""), cve.ID("cve2", "")}),
							ImageComponentId: scancomponent.ComponentID("comp1", "ver2", ""),
							ImageCveId:       cve.ID("cve2", ""),
							FixedBy:          proto.String("ver3"),
							IsFixable:        true,
						}.Build(),
						CVEV2: storage.ImageCVEV2_builder{
							Id:      getTestCVEID(testCVEs["cve2comp1"], getTestComponentID(testComponents[1], "sha")),
							ImageId: "sha",
							CveBaseInfo: storage.CVEInfo_builder{
								Cve:       "cve2",
								CreatedAt: ts,
							}.Build(),
							NvdScoreVersion:      storage.CvssScoreVersion_UNKNOWN_VERSION,
							FixedBy:              proto.String("ver3"),
							IsFixable:            true,
							FirstImageOccurrence: ts,
							ComponentId:          getTestComponentID(testComponents[1], "sha"),
						}.Build(),
					},
				},
			},
			{
				Component: storage.ImageComponent_builder{
					Id:      scancomponent.ComponentID("comp2", "ver1", ""),
					Name:    "comp2",
					Version: "ver1",
				}.Build(),
				ComponentV2: storage.ImageComponentV2_builder{
					Id:         getTestComponentID(testComponents[2], "sha"),
					Name:       "comp2",
					Version:    "ver1",
					ImageId:    "sha",
					LayerIndex: proto.Int32(2),
				}.Build(),
				Edge: storage.ImageComponentEdge_builder{
					Id:               pgSearch.IDFromPks([]string{"sha", scancomponent.ComponentID("comp2", "ver1", "")}),
					ImageId:          "sha",
					ImageComponentId: scancomponent.ComponentID("comp2", "ver1", ""),
					LayerIndex:       proto.Int32(2),
				}.Build(),
				Children: []CVEParts{
					{
						CVE: storage.ImageCVE_builder{
							Id: cve.ID("cve1", ""),
							CveBaseInfo: storage.CVEInfo_builder{
								Cve:       "cve1",
								CreatedAt: ts,
							}.Build(),
							NvdScoreVersion: storage.CvssScoreVersion_UNKNOWN_VERSION,
						}.Build(),
						Edge: storage.ComponentCVEEdge_builder{
							Id:               pgSearch.IDFromPks([]string{scancomponent.ComponentID("comp2", "ver1", ""), cve.ID("cve1", "")}),
							ImageComponentId: scancomponent.ComponentID("comp2", "ver1", ""),
							ImageCveId:       cve.ID("cve1", ""),
							FixedBy:          proto.String("ver2"),
							IsFixable:        true,
						}.Build(),
						CVEV2: storage.ImageCVEV2_builder{
							Id:      getTestCVEID(testCVEs["cve1comp2"], getTestComponentID(testComponents[2], "sha")),
							ImageId: "sha",
							CveBaseInfo: storage.CVEInfo_builder{
								Cve:       "cve1",
								CreatedAt: ts,
							}.Build(),
							NvdScoreVersion:      storage.CvssScoreVersion_UNKNOWN_VERSION,
							FixedBy:              proto.String("ver2"),
							IsFixable:            true,
							FirstImageOccurrence: ts,
							ComponentId:          getTestComponentID(testComponents[2], "sha"),
						}.Build(),
					},
					{
						CVE: storage.ImageCVE_builder{
							Id: cve.ID("cve2", ""),
							CveBaseInfo: storage.CVEInfo_builder{
								Cve:       "cve2",
								CreatedAt: ts,
							}.Build(),
							NvdScoreVersion: storage.CvssScoreVersion_UNKNOWN_VERSION,
						}.Build(),
						Edge: storage.ComponentCVEEdge_builder{
							Id:               pgSearch.IDFromPks([]string{scancomponent.ComponentID("comp2", "ver1", ""), cve.ID("cve2", "")}),
							ImageComponentId: scancomponent.ComponentID("comp2", "ver1", ""),
							ImageCveId:       cve.ID("cve2", ""),
						}.Build(),
						CVEV2: storage.ImageCVEV2_builder{
							Id:      getTestCVEID(testCVEs["cve2comp2"], getTestComponentID(testComponents[2], "sha")),
							ImageId: "sha",
							CveBaseInfo: storage.CVEInfo_builder{
								Cve:       "cve2",
								CreatedAt: ts,
							}.Build(),
							NvdScoreVersion:      storage.CvssScoreVersion_UNKNOWN_VERSION,
							FirstImageOccurrence: ts,
							ComponentId:          getTestComponentID(testComponents[2], "sha"),
						}.Build(),
					},
				},
			},
		},
	}

	splitActual, err := SplitV2(image, true)
	assert.NoError(t, err)
	if !features.FlattenCVEData.Enabled() {
		protoassert.MapEqual(t, splitExpected.ImageCVEEdges, splitActual.ImageCVEEdges)
	}
	protoassert.Equal(t, splitExpected.Image, splitActual.Image)

	assert.Len(t, splitActual.Children, len(splitExpected.Children))
	for i, expected := range splitExpected.Children {
		actual := splitActual.Children[i]
		if features.FlattenCVEData.Enabled() {
			protoassert.Equal(t, expected.ComponentV2, actual.ComponentV2)
		} else {
			protoassert.Equal(t, expected.Component, actual.Component)
			protoassert.Equal(t, expected.Edge, actual.Edge)
		}

		assert.Len(t, actual.Children, len(expected.Children))
		for i, e := range expected.Children {
			a := actual.Children[i]
			if features.FlattenCVEData.Enabled() {
				protoassert.Equal(t, e.CVEV2, a.CVEV2)
			} else {
				protoassert.Equal(t, e.Edge, a.Edge)
				protoassert.Equal(t, e.CVE, a.CVE)
			}
		}
	}

	// Need to add first occurrence edges as otherwise they will be filtered out
	// These values are added on insertion for the DB which is why we will populate them artificially here
	for _, v := range splitActual.ImageCVEEdges {
		v.SetFirstImageOccurrence(ts)
	}

	var imageActual *storage.Image
	if features.FlattenCVEData.Enabled() {
		imageActual = MergeV2(splitActual)
	} else {
		imageActual = Merge(splitActual)
	}
	protoassert.Equal(t, dedupedImage(), imageActual)
}

func getTestComponentID(testComponent *storage.EmbeddedImageScanComponent, imageID string) string {
	id, _ := scancomponent.ComponentIDV2(testComponent, imageID)

	return id
}

func getTestCVEID(testCVE *storage.EmbeddedVulnerability, componentID string) string {
	id, _ := cve.IDV2(testCVE, componentID)

	return id
}

func dedupedImage() *storage.Image {
	return storage.Image_builder{
		Id: "sha",
		Name: storage.ImageName_builder{
			FullName: "name",
		}.Build(),
		Metadata: storage.ImageMetadata_builder{
			V1: storage.V1Metadata_builder{
				Created: ts,
			}.Build(),
		}.Build(),
		Components:  proto.Int32(3),
		Cves:        proto.Int32(4),
		FixableCves: proto.Int32(2),
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
