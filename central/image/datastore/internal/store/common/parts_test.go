package common

import (
	"testing"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/edges"
	"github.com/stackrox/stackrox/pkg/scancomponent"
	"github.com/stretchr/testify/assert"
)

func TestSplitAndMergeImage(t *testing.T) {
	ts := timestamp.TimestampNow()
	image := &storage.Image{
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
			Components: 3,
		},
		SetCves: &storage.Image_Cves{
			Cves: 4,
		},
		SetFixable: &storage.Image_FixableCves{
			FixableCves: 2,
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
							Cve:                  "cve1",
							VulnerabilityType:    storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							FirstImageOccurrence: ts,
						},
						{
							Cve:               "cve2",
							VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "ver3",
							},
							FirstImageOccurrence: ts,
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
							Cve:               "cve1",
							VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "ver2",
							},
							FirstImageOccurrence: ts,
						},
						{
							Cve:                  "cve2",
							VulnerabilityType:    storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							FirstImageOccurrence: ts,
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
			Scan: &storage.ImageScan{
				ScanTime: ts,
			},
			SetComponents: &storage.Image_Components{
				Components: 3,
			},
			SetCves: &storage.Image_Cves{
				Cves: 4,
			},
			SetFixable: &storage.Image_FixableCves{
				FixableCves: 2,
			},
		},
		ListImage: &storage.ListImage{
			Id:      "sha",
			Name:    "name",
			Created: ts,
			SetComponents: &storage.ListImage_Components{
				Components: 3,
			},
			SetCves: &storage.ListImage_Cves{
				Cves: 4,
			},
			SetFixable: &storage.ListImage_FixableCves{
				FixableCves: 2,
			},
		},
		ImageCVEEdges: map[string]*storage.ImageCVEEdge{
			"cve1": {
				Id:         edges.EdgeID{ParentID: "sha", ChildID: "cve1"}.ToString(),
				ImageId:    "sha",
				ImageCveId: "cve1",
			},
			"cve2": {
				Id:         edges.EdgeID{ParentID: "sha", ChildID: "cve2"}.ToString(),
				ImageId:    "sha",
				ImageCveId: "cve2",
			},
		},
		Children: []ComponentParts{
			{
				Component: &storage.ImageComponent{
					Id:      scancomponent.ComponentID("comp1", "ver1", ""),
					Name:    "comp1",
					Version: "ver1",
				},
				Edge: &storage.ImageComponentEdge{
					Id:               edges.EdgeID{ParentID: "sha", ChildID: scancomponent.ComponentID("comp1", "ver1", "")}.ToString(),
					ImageId:          "sha",
					ImageComponentId: scancomponent.ComponentID("comp1", "ver1", ""),
					HasLayerIndex: &storage.ImageComponentEdge_LayerIndex{
						LayerIndex: 1,
					},
				},
				Children: []CVEParts{},
			},
			{
				Component: &storage.ImageComponent{
					Id:      scancomponent.ComponentID("comp1", "ver2", ""),
					Name:    "comp1",
					Version: "ver2",
				},
				Edge: &storage.ImageComponentEdge{
					Id:               edges.EdgeID{ParentID: "sha", ChildID: scancomponent.ComponentID("comp1", "ver2", "")}.ToString(),
					ImageId:          "sha",
					ImageComponentId: scancomponent.ComponentID("comp1", "ver2", ""),
					HasLayerIndex: &storage.ImageComponentEdge_LayerIndex{
						LayerIndex: 3,
					},
				},
				Children: []CVEParts{
					{
						Cve: &storage.CVE{
							Id:   "cve1",
							Cve:  "cve1",
							Type: storage.CVE_IMAGE_CVE,
						},
						Edge: &storage.ComponentCVEEdge{
							Id:               edges.EdgeID{ParentID: scancomponent.ComponentID("comp1", "ver2", ""), ChildID: "cve1"}.ToString(),
							ImageComponentId: scancomponent.ComponentID("comp1", "ver2", ""),
							ImageCveId:       "cve1",
						},
					},
					{
						Cve: &storage.CVE{
							Id:   "cve2",
							Cve:  "cve2",
							Type: storage.CVE_IMAGE_CVE,
						},
						Edge: &storage.ComponentCVEEdge{
							Id:               edges.EdgeID{ParentID: scancomponent.ComponentID("comp1", "ver2", ""), ChildID: "cve2"}.ToString(),
							ImageComponentId: scancomponent.ComponentID("comp1", "ver2", ""),
							ImageCveId:       "cve2",
							HasFixedBy: &storage.ComponentCVEEdge_FixedBy{
								FixedBy: "ver3",
							},
							IsFixable: true,
						},
					},
				},
			},
			{
				Component: &storage.ImageComponent{
					Id:      scancomponent.ComponentID("comp2", "ver1", ""),
					Name:    "comp2",
					Version: "ver1",
				},
				Edge: &storage.ImageComponentEdge{
					Id:               edges.EdgeID{ParentID: "sha", ChildID: scancomponent.ComponentID("comp2", "ver1", "")}.ToString(),
					ImageId:          "sha",
					ImageComponentId: scancomponent.ComponentID("comp2", "ver1", ""),
					HasLayerIndex: &storage.ImageComponentEdge_LayerIndex{
						LayerIndex: 2,
					},
				},
				Children: []CVEParts{
					{
						Cve: &storage.CVE{
							Id:   "cve1",
							Cve:  "cve1",
							Type: storage.CVE_IMAGE_CVE,
						},
						Edge: &storage.ComponentCVEEdge{
							Id:               edges.EdgeID{ParentID: scancomponent.ComponentID("comp2", "ver1", ""), ChildID: "cve1"}.ToString(),
							ImageComponentId: scancomponent.ComponentID("comp2", "ver1", ""),
							ImageCveId:       "cve1",
							HasFixedBy: &storage.ComponentCVEEdge_FixedBy{
								FixedBy: "ver2",
							},
							IsFixable: true,
						},
					},
					{
						Cve: &storage.CVE{
							Id:   "cve2",
							Cve:  "cve2",
							Type: storage.CVE_IMAGE_CVE,
						},
						Edge: &storage.ComponentCVEEdge{
							Id:               edges.EdgeID{ParentID: scancomponent.ComponentID("comp2", "ver1", ""), ChildID: "cve2"}.ToString(),
							ImageComponentId: scancomponent.ComponentID("comp2", "ver1", ""),
							ImageCveId:       "cve2",
						},
					},
				},
			},
		},
	}

	splitActual := Split(image, true)
	assert.Equal(t, splitExpected, splitActual)

	// Need to add first occurrence edges as otherwise they will be filtered out
	// These values are added on insertion for the DB which is why we will populate them artificially here
	for _, v := range splitActual.ImageCVEEdges {
		v.FirstImageOccurrence = ts
	}

	imageActual := Merge(splitActual)
	assert.Equal(t, image, imageActual)
}
