package dackbox

import (
	"testing"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stackrox/rox/pkg/scancomponent"
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
							Cve:               "cve1",
							VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
						},
						{
							Cve:               "cve2",
							VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "ver3",
							},
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
						},
						{
							Cve:               "cve2",
							VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
						},
					},
				},
			},
		},
	}

	splitExpected := ImageParts{
		image: &storage.Image{
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
		listImage: &storage.ListImage{
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
		imageCVEEdges: map[string]*storage.ImageCVEEdge{
			"cve1": {
				Id: edges.EdgeID{ParentID: "sha", ChildID: "cve1"}.ToString(),
			},
			"cve2": {
				Id: edges.EdgeID{ParentID: "sha", ChildID: "cve2"}.ToString(),
			},
		},
		children: []ComponentParts{
			{
				component: &storage.ImageComponent{
					Id:      scancomponent.ComponentID("comp1", "ver1"),
					Name:    "comp1",
					Version: "ver1",
				},
				edge: &storage.ImageComponentEdge{
					Id: edges.EdgeID{ParentID: "sha", ChildID: scancomponent.ComponentID("comp1", "ver1")}.ToString(),
					HasLayerIndex: &storage.ImageComponentEdge_LayerIndex{
						LayerIndex: 1,
					},
				},
				children: []CVEParts{},
			},
			{
				component: &storage.ImageComponent{
					Id:      scancomponent.ComponentID("comp1", "ver2"),
					Name:    "comp1",
					Version: "ver2",
				},
				edge: &storage.ImageComponentEdge{
					Id: edges.EdgeID{ParentID: "sha", ChildID: scancomponent.ComponentID("comp1", "ver2")}.ToString(),
					HasLayerIndex: &storage.ImageComponentEdge_LayerIndex{
						LayerIndex: 3,
					},
				},
				children: []CVEParts{
					{
						cve: &storage.CVE{
							Id:   "cve1",
							Type: storage.CVE_IMAGE_CVE,
						},
						edge: &storage.ComponentCVEEdge{
							Id: edges.EdgeID{ParentID: scancomponent.ComponentID("comp1", "ver2"), ChildID: "cve1"}.ToString(),
						},
					},
					{
						cve: &storage.CVE{
							Id:   "cve2",
							Type: storage.CVE_IMAGE_CVE,
						},
						edge: &storage.ComponentCVEEdge{
							Id: edges.EdgeID{ParentID: scancomponent.ComponentID("comp1", "ver2"), ChildID: "cve2"}.ToString(),
							HasFixedBy: &storage.ComponentCVEEdge_FixedBy{
								FixedBy: "ver3",
							},
							IsFixable: true,
						},
					},
				},
			},
			{
				component: &storage.ImageComponent{
					Id:      scancomponent.ComponentID("comp2", "ver1"),
					Name:    "comp2",
					Version: "ver1",
				},
				edge: &storage.ImageComponentEdge{
					Id: edges.EdgeID{ParentID: "sha", ChildID: scancomponent.ComponentID("comp2", "ver1")}.ToString(),
					HasLayerIndex: &storage.ImageComponentEdge_LayerIndex{
						LayerIndex: 2,
					},
				},
				children: []CVEParts{
					{
						cve: &storage.CVE{
							Id:   "cve1",
							Type: storage.CVE_IMAGE_CVE,
						},
						edge: &storage.ComponentCVEEdge{
							Id: edges.EdgeID{ParentID: scancomponent.ComponentID("comp2", "ver1"), ChildID: "cve1"}.ToString(),
							HasFixedBy: &storage.ComponentCVEEdge_FixedBy{
								FixedBy: "ver2",
							},
							IsFixable: true,
						},
					},
					{
						cve: &storage.CVE{
							Id:   "cve2",
							Type: storage.CVE_IMAGE_CVE,
						},
						edge: &storage.ComponentCVEEdge{
							Id: edges.EdgeID{ParentID: scancomponent.ComponentID("comp2", "ver1"), ChildID: "cve2"}.ToString(),
						},
					},
				},
			},
		},
	}

	splitActual := Split(image, true)
	assert.Equal(t, splitExpected, splitActual)

	imageActual := Merge(splitActual)
	assert.Equal(t, image, imageActual)
}
