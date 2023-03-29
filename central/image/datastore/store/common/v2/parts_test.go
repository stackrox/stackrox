package common

import (
	"testing"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/scancomponent"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
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
							VulnerabilityTypes:   []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							FirstImageOccurrence: ts,
						},
						{
							Cve:                "cve2",
							VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
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
							Cve:                "cve1",
							VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "ver2",
							},
							FirstImageOccurrence: ts,
						},
						{
							Cve:                  "cve2",
							VulnerabilityType:    storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes:   []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
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
		ImageCVEEdges: map[string]*storage.ImageCVEEdge{
			cve.ID("cve1", ""): {
				Id:         pgSearch.IDFromPks([]string{"sha", cve.ID("cve1", "")}),
				ImageId:    "sha",
				ImageCveId: cve.ID("cve1", ""),
			},
			cve.ID("cve2", ""): {
				Id:         pgSearch.IDFromPks([]string{"sha", cve.ID("cve2", "")}),
				ImageId:    "sha",
				ImageCveId: cve.ID("cve2", ""),
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
					Id:               pgSearch.IDFromPks([]string{"sha", scancomponent.ComponentID("comp1", "ver1", "")}),
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
					Id:               pgSearch.IDFromPks([]string{"sha", scancomponent.ComponentID("comp1", "ver2", "")}),
					ImageId:          "sha",
					ImageComponentId: scancomponent.ComponentID("comp1", "ver2", ""),
					HasLayerIndex: &storage.ImageComponentEdge_LayerIndex{
						LayerIndex: 3,
					},
				},
				Children: []CVEParts{
					{
						CVE: &storage.ImageCVE{
							Id: cve.ID("cve1", ""),
							CveBaseInfo: &storage.CVEInfo{
								Cve: "cve1",
							},
						},
						Edge: &storage.ComponentCVEEdge{
							Id:               pgSearch.IDFromPks([]string{scancomponent.ComponentID("comp1", "ver2", ""), cve.ID("cve1", "")}),
							ImageComponentId: scancomponent.ComponentID("comp1", "ver2", ""),
							ImageCveId:       cve.ID("cve1", ""),
						},
					},
					{
						CVE: &storage.ImageCVE{
							Id: cve.ID("cve2", ""),
							CveBaseInfo: &storage.CVEInfo{
								Cve: "cve2",
							},
						},
						Edge: &storage.ComponentCVEEdge{
							Id:               pgSearch.IDFromPks([]string{scancomponent.ComponentID("comp1", "ver2", ""), cve.ID("cve2", "")}),
							ImageComponentId: scancomponent.ComponentID("comp1", "ver2", ""),
							ImageCveId:       cve.ID("cve2", ""),
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
					Id:               pgSearch.IDFromPks([]string{"sha", scancomponent.ComponentID("comp2", "ver1", "")}),
					ImageId:          "sha",
					ImageComponentId: scancomponent.ComponentID("comp2", "ver1", ""),
					HasLayerIndex: &storage.ImageComponentEdge_LayerIndex{
						LayerIndex: 2,
					},
				},
				Children: []CVEParts{
					{
						CVE: &storage.ImageCVE{
							Id: cve.ID("cve1", ""),
							CveBaseInfo: &storage.CVEInfo{
								Cve: "cve1",
							},
						},
						Edge: &storage.ComponentCVEEdge{
							Id:               pgSearch.IDFromPks([]string{scancomponent.ComponentID("comp2", "ver1", ""), cve.ID("cve1", "")}),
							ImageComponentId: scancomponent.ComponentID("comp2", "ver1", ""),
							ImageCveId:       cve.ID("cve1", ""),
							HasFixedBy: &storage.ComponentCVEEdge_FixedBy{
								FixedBy: "ver2",
							},
							IsFixable: true,
						},
					},
					{
						CVE: &storage.ImageCVE{
							Id: cve.ID("cve2", ""),
							CveBaseInfo: &storage.CVEInfo{
								Cve: "cve2",
							},
						},
						Edge: &storage.ComponentCVEEdge{
							Id:               pgSearch.IDFromPks([]string{scancomponent.ComponentID("comp2", "ver1", ""), cve.ID("cve2", "")}),
							ImageComponentId: scancomponent.ComponentID("comp2", "ver1", ""),
							ImageCveId:       cve.ID("cve2", ""),
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
