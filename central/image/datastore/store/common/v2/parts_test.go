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
				ComponentV2: &storage.ImageComponentV2{
					Id:      getTestComponentID(testComponents[0], "sha"),
					Name:    "comp1",
					Version: "ver1",
					ImageId: "sha",
					HasLayerIndex: &storage.ImageComponentV2_LayerIndex{
						LayerIndex: 1,
					},
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
				ComponentV2: &storage.ImageComponentV2{
					Id:      getTestComponentID(testComponents[1], "sha"),
					Name:    "comp1",
					Version: "ver2",
					ImageId: "sha",
					HasLayerIndex: &storage.ImageComponentV2_LayerIndex{
						LayerIndex: 3,
					},
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
								Cve:       "cve1",
								CreatedAt: ts,
							},
							NvdScoreVersion: storage.CvssScoreVersion_UNKNOWN_VERSION,
						},
						Edge: &storage.ComponentCVEEdge{
							Id:               pgSearch.IDFromPks([]string{scancomponent.ComponentID("comp1", "ver2", ""), cve.ID("cve1", "")}),
							ImageComponentId: scancomponent.ComponentID("comp1", "ver2", ""),
							ImageCveId:       cve.ID("cve1", ""),
						},
						CVEV2: &storage.ImageCVEV2{
							Id:      getTestCVEID(testCVEs["cve1comp1"], getTestComponentID(testComponents[1], "sha")),
							ImageId: "sha",
							CveBaseInfo: &storage.CVEInfo{
								Cve:       "cve1",
								CreatedAt: ts,
							},
							NvdScoreVersion:      storage.CvssScoreVersion_UNKNOWN_VERSION,
							FirstImageOccurrence: ts,
							ComponentId:          getTestComponentID(testComponents[1], "sha"),
						},
					},
					{
						CVE: &storage.ImageCVE{
							Id: cve.ID("cve2", ""),
							CveBaseInfo: &storage.CVEInfo{
								Cve:       "cve2",
								CreatedAt: ts,
							},
							NvdScoreVersion: storage.CvssScoreVersion_UNKNOWN_VERSION,
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
						CVEV2: &storage.ImageCVEV2{
							Id:      getTestCVEID(testCVEs["cve2comp1"], getTestComponentID(testComponents[1], "sha")),
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
							ComponentId:          getTestComponentID(testComponents[1], "sha"),
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
				ComponentV2: &storage.ImageComponentV2{
					Id:      getTestComponentID(testComponents[2], "sha"),
					Name:    "comp2",
					Version: "ver1",
					ImageId: "sha",
					HasLayerIndex: &storage.ImageComponentV2_LayerIndex{
						LayerIndex: 2,
					},
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
								Cve:       "cve1",
								CreatedAt: ts,
							},
							NvdScoreVersion: storage.CvssScoreVersion_UNKNOWN_VERSION,
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
						CVEV2: &storage.ImageCVEV2{
							Id:      getTestCVEID(testCVEs["cve1comp2"], getTestComponentID(testComponents[2], "sha")),
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
							ComponentId:          getTestComponentID(testComponents[2], "sha"),
						},
					},
					{
						CVE: &storage.ImageCVE{
							Id: cve.ID("cve2", ""),
							CveBaseInfo: &storage.CVEInfo{
								Cve:       "cve2",
								CreatedAt: ts,
							},
							NvdScoreVersion: storage.CvssScoreVersion_UNKNOWN_VERSION,
						},
						Edge: &storage.ComponentCVEEdge{
							Id:               pgSearch.IDFromPks([]string{scancomponent.ComponentID("comp2", "ver1", ""), cve.ID("cve2", "")}),
							ImageComponentId: scancomponent.ComponentID("comp2", "ver1", ""),
							ImageCveId:       cve.ID("cve2", ""),
						},
						CVEV2: &storage.ImageCVEV2{
							Id:      getTestCVEID(testCVEs["cve2comp2"], getTestComponentID(testComponents[2], "sha")),
							ImageId: "sha",
							CveBaseInfo: &storage.CVEInfo{
								Cve:       "cve2",
								CreatedAt: ts,
							},
							NvdScoreVersion:      storage.CvssScoreVersion_UNKNOWN_VERSION,
							FirstImageOccurrence: ts,
							ComponentId:          getTestComponentID(testComponents[2], "sha"),
						},
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
		v.FirstImageOccurrence = ts
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
