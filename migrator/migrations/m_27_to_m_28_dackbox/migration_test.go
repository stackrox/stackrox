package m27tom28

import (
	"testing"

	"github.com/dgraph-io/badger"
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/badgerhelpers"
	"github.com/stretchr/testify/suite"
)

func TestDackBoxMigration(t *testing.T) {
	suite.Run(t, new(DackBoxMigrationTestSuite))
}

type DackBoxMigrationTestSuite struct {
	suite.Suite

	db *badger.DB
}

func (suite *DackBoxMigrationTestSuite) SetupSuite() {
	var err error
	suite.db, err = badgerhelpers.NewTemp("dackbox_migration_test")
	if err != nil {
		suite.FailNowf("failed to create DB: %+v", err.Error())
	}
}

func (suite *DackBoxMigrationTestSuite) TearDownSuite() {
	_ = suite.db.Close()
}

func (suite *DackBoxMigrationTestSuite) TestImages() {
	ts := timestamp.TimestampNow()
	deployments := []*storage.Deployment{
		{
			ClusterId: "cid1",
			Namespace: "ns1",
			Id:        "did1",
			Name:      "foo",
			Type:      "Replicated",
			Created:   ts,
			Containers: []*storage.Container{
				{
					Image: &storage.ContainerImage{
						Id: "sha2",
					},
				},
				{
					Image: &storage.ContainerImage{
						Id: "sha1",
					},
				},
			},
		},
		{
			ClusterId: "cid1",
			Namespace: "ns2",
			Id:        "did2",
			Name:      "bar",
			Type:      "Global",
			Created:   ts,
		},
	}

	images := []*storage.Image{
		{
			Id: "sha1",
			Name: &storage.ImageName{
				FullName: "name1",
			},
			Metadata: &storage.ImageMetadata{
				V1: &storage.V1Metadata{
					Created: ts,
				},
			},
			Scan: &storage.ImageScan{
				ScanTime: ts,
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Name:    "comp1",
						Version: "ver1",
						HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{
							LayerIndex: 2,
						},
						Vulns: []*storage.EmbeddedVulnerability{},
					},
					{
						Name:    "comp1",
						Version: "ver2",
						HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{
							LayerIndex: 1,
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
							LayerIndex: 3,
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
		},
		{
			Id: "sha2",
			Name: &storage.ImageName{
				FullName: "name2",
			},
			Metadata: &storage.ImageMetadata{
				V1: &storage.V1Metadata{
					Created: ts,
				},
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
							LayerIndex: 2,
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
						Name:    "comp3",
						Version: "ver1",
						HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{
							LayerIndex: 3,
						},
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve:               "cve3",
								VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
								SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
									FixedBy: "ver1",
								},
							},
							{
								Cve:               "cve4",
								VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							},
						},
					},
				},
			},
		},
	}

	slimmedImages := []*storage.Image{
		{
			Id: "sha1",
			Name: &storage.ImageName{
				FullName: "name1",
			},
			Metadata: &storage.ImageMetadata{
				V1: &storage.V1Metadata{
					Created: ts,
				},
			},
			Scan: &storage.ImageScan{
				ScanTime: ts,
			},
		},
		{
			Id: "sha2",
			Name: &storage.ImageName{
				FullName: "name2",
			},
			Metadata: &storage.ImageMetadata{
				V1: &storage.V1Metadata{
					Created: ts,
				},
			},
			Scan: &storage.ImageScan{
				ScanTime: ts,
			},
		},
	}

	components := []*storage.ImageComponent{
		{
			Id:      encodeIDPair("comp1", "ver1"),
			Name:    "comp1",
			Version: "ver1",
		},
		{
			Id:      encodeIDPair("comp1", "ver2"),
			Name:    "comp1",
			Version: "ver2",
		},
		{
			Id:      encodeIDPair("comp2", "ver1"),
			Name:    "comp2",
			Version: "ver1",
		},
		{
			Id:      encodeIDPair("comp3", "ver1"),
			Name:    "comp3",
			Version: "ver1",
		},
	}

	componentEdges := []*storage.ImageComponentEdge{
		{
			Id: encodeIDPair(images[0].GetId(), components[0].GetId()),
			HasLayerIndex: &storage.ImageComponentEdge_LayerIndex{
				LayerIndex: 2,
			},
		},
		{
			Id: encodeIDPair(images[0].GetId(), components[1].GetId()),
			HasLayerIndex: &storage.ImageComponentEdge_LayerIndex{
				LayerIndex: 1,
			},
		},
		{
			Id: encodeIDPair(images[0].GetId(), components[2].GetId()),
			HasLayerIndex: &storage.ImageComponentEdge_LayerIndex{
				LayerIndex: 3,
			},
		},
		{
			Id: encodeIDPair(images[1].GetId(), components[0].GetId()),
			HasLayerIndex: &storage.ImageComponentEdge_LayerIndex{
				LayerIndex: 1,
			},
		},
		{
			Id: encodeIDPair(images[1].GetId(), components[1].GetId()),
			HasLayerIndex: &storage.ImageComponentEdge_LayerIndex{
				LayerIndex: 2,
			},
		},
		{
			Id: encodeIDPair(images[1].GetId(), components[3].GetId()),
			HasLayerIndex: &storage.ImageComponentEdge_LayerIndex{
				LayerIndex: 3,
			},
		},
	}

	cves := []*storage.CVE{
		{
			Id:   "cve1",
			Type: storage.CVE_IMAGE_CVE,
		},
		{
			Id:   "cve2",
			Type: storage.CVE_IMAGE_CVE,
		},
		{
			Id:   "cve3",
			Type: storage.CVE_IMAGE_CVE,
		},
		{
			Id:   "cve4",
			Type: storage.CVE_IMAGE_CVE,
		},
	}

	cveEdges := []*storage.ComponentCVEEdge{
		{
			Id: encodeIDPair(components[1].GetId(), cves[0].GetId()),
		},
		{
			Id: encodeIDPair(components[1].GetId(), cves[1].GetId()),
			HasFixedBy: &storage.ComponentCVEEdge_FixedBy{
				FixedBy: "ver3",
			},
			IsFixable: true,
		},
		{
			Id: encodeIDPair(components[2].GetId(), cves[0].GetId()),
			HasFixedBy: &storage.ComponentCVEEdge_FixedBy{
				FixedBy: "ver2",
			},
			IsFixable: true,
		},
		{
			Id: encodeIDPair(components[2].GetId(), cves[1].GetId()),
		},
		{
			Id: encodeIDPair(components[3].GetId(), cves[2].GetId()),
			HasFixedBy: &storage.ComponentCVEEdge_FixedBy{
				FixedBy: "ver1",
			},
			IsFixable: true,
		},
		{
			Id: encodeIDPair(components[3].GetId(), cves[3].GetId()),
		},
	}

	// Write old versions of the deployments and images.
	batch := suite.db.NewWriteBatch()
	defer batch.Cancel()
	err := writeProto(batch, getDeploymentKey(deployments[0].GetId()), deployments[0])
	suite.NoError(err)
	err = writeProto(batch, getDeploymentKey(deployments[1].GetId()), deployments[1])
	suite.NoError(err)
	err = writeProto(batch, getImageKey(images[0].GetId()), images[0])
	suite.NoError(err)
	err = writeProto(batch, getImageKey(images[1].GetId()), images[1])
	suite.NoError(err)
	err = batch.Flush()
	suite.NoError(err)

	keys, err := getKeysWithPrefix(deploymentBucketName, suite.db)
	suite.NoError(err)
	suite.Equal([][]byte{getDeploymentKey("did1"), getDeploymentKey("did2")}, keys)

	// Run the migration.
	err = migrateDeploymentsAndImages(suite.db)
	suite.NoError(err)

	// Check that the deployments haven't changed.
	readDeployment := &storage.Deployment{}
	_, err = readProto(suite.db, getDeploymentKey(deployments[0].GetId()), readDeployment)
	suite.NoError(err)
	suite.Equal(deployments[0], readDeployment)
	_, err = readProto(suite.db, getDeploymentKey(deployments[1].GetId()), readDeployment)
	suite.NoError(err)
	suite.Equal(deployments[1], readDeployment)

	// Check that the new images are slimmed down like we expect.
	readImage := &storage.Image{}
	_, err = readProto(suite.db, getImageKey(images[0].GetId()), readImage)
	suite.NoError(err)
	suite.Equal(slimmedImages[0], readImage)
	_, err = readProto(suite.db, getImageKey(images[1].GetId()), readImage)
	suite.NoError(err)
	suite.Equal(slimmedImages[1], readImage)

	// Check that the components were recorded as expected.
	keys, err = getKeysWithPrefix(componentsBucketName, suite.db)
	suite.NoError(err)
	suite.Equal(4, len(keys))

	readComponent := &storage.ImageComponent{}
	_, err = readProto(suite.db, getComponentKey(components[0].GetId()), readComponent)
	suite.NoError(err)
	suite.Equal(components[0], readComponent)
	_, err = readProto(suite.db, getComponentKey(components[1].GetId()), readComponent)
	suite.NoError(err)
	suite.Equal(components[1], readComponent)
	_, err = readProto(suite.db, getComponentKey(components[2].GetId()), readComponent)
	suite.NoError(err)
	suite.Equal(components[2], readComponent)
	_, err = readProto(suite.db, getComponentKey(components[3].GetId()), readComponent)
	suite.NoError(err)
	suite.Equal(components[3], readComponent)

	// Check that the image component edges were added correctly
	keys, err = getKeysWithPrefix(imageToComponentsBucketName, suite.db)
	suite.NoError(err)
	suite.Equal(6, len(keys))

	imageComponentEdge := &storage.ImageComponentEdge{}
	_, err = readProto(suite.db, getImageComponentEdgeKey(componentEdges[0].GetId()), imageComponentEdge)
	suite.NoError(err)
	suite.Equal(componentEdges[0], imageComponentEdge)
	_, err = readProto(suite.db, getImageComponentEdgeKey(componentEdges[1].GetId()), imageComponentEdge)
	suite.NoError(err)
	suite.Equal(componentEdges[1], imageComponentEdge)
	_, err = readProto(suite.db, getImageComponentEdgeKey(componentEdges[2].GetId()), imageComponentEdge)
	suite.NoError(err)
	suite.Equal(componentEdges[2], imageComponentEdge)
	_, err = readProto(suite.db, getImageComponentEdgeKey(componentEdges[3].GetId()), imageComponentEdge)
	suite.NoError(err)
	suite.Equal(componentEdges[3], imageComponentEdge)
	_, err = readProto(suite.db, getImageComponentEdgeKey(componentEdges[4].GetId()), imageComponentEdge)
	suite.NoError(err)
	suite.Equal(componentEdges[4], imageComponentEdge)
	_, err = readProto(suite.db, getImageComponentEdgeKey(componentEdges[5].GetId()), imageComponentEdge)
	suite.NoError(err)
	suite.Equal(componentEdges[5], imageComponentEdge)

	// Check that the cves were added correctly.
	keys, err = getKeysWithPrefix(cveBucketName, suite.db)
	suite.NoError(err)
	suite.Equal(4, len(keys))

	readCVE := &storage.CVE{}
	_, err = readProto(suite.db, getCVEKey(cves[0].GetId()), readCVE)
	suite.NoError(err)
	suite.Equal(cves[0], readCVE)
	_, err = readProto(suite.db, getCVEKey(cves[1].GetId()), readCVE)
	suite.NoError(err)
	suite.Equal(cves[1], readCVE)
	_, err = readProto(suite.db, getCVEKey(cves[2].GetId()), readCVE)
	suite.NoError(err)
	suite.Equal(cves[2], readCVE)
	_, err = readProto(suite.db, getCVEKey(cves[3].GetId()), readCVE)
	suite.NoError(err)
	suite.Equal(cves[3], readCVE)

	// Check that the component-cve edges were added correclty.
	keys, err = getKeysWithPrefix(imageToComponentsBucketName, suite.db)
	suite.NoError(err)
	suite.Equal(6, len(keys))

	readComponentCVEEdge := &storage.ComponentCVEEdge{}
	_, err = readProto(suite.db, getComponentCVEEdgeKey(cveEdges[0].GetId()), readComponentCVEEdge)
	suite.NoError(err)
	suite.Equal(cveEdges[0], readComponentCVEEdge)
	_, err = readProto(suite.db, getComponentCVEEdgeKey(cveEdges[1].GetId()), readComponentCVEEdge)
	suite.NoError(err)
	suite.Equal(cveEdges[1], readComponentCVEEdge)
	_, err = readProto(suite.db, getComponentCVEEdgeKey(cveEdges[2].GetId()), readComponentCVEEdge)
	suite.NoError(err)
	suite.Equal(cveEdges[2], readComponentCVEEdge)
	_, err = readProto(suite.db, getComponentCVEEdgeKey(cveEdges[3].GetId()), readComponentCVEEdge)
	suite.NoError(err)
	suite.Equal(cveEdges[3], readComponentCVEEdge)
	_, err = readProto(suite.db, getComponentCVEEdgeKey(cveEdges[4].GetId()), readComponentCVEEdge)
	suite.NoError(err)
	suite.Equal(cveEdges[4], readComponentCVEEdge)
	_, err = readProto(suite.db, getComponentCVEEdgeKey(cveEdges[5].GetId()), readComponentCVEEdge)
	suite.NoError(err)
	suite.Equal(cveEdges[5], readComponentCVEEdge)

	// Check that the graph is what we expect.
	tos, err := readMapping(suite.db, getClusterKey("cid1"))
	suite.NoError(err)
	suite.Equal(SortedKeys{getNamespaceKey("ns1"), getNamespaceKey("ns2")}, tos)

	tos, err = readMapping(suite.db, getNamespaceKey("ns1"))
	suite.NoError(err)
	suite.Equal(SortedKeys{getDeploymentKey(deployments[0].GetId())}, tos)

	tos, err = readMapping(suite.db, getNamespaceKey("ns2"))
	suite.NoError(err)
	suite.Equal(SortedKeys{getDeploymentKey(deployments[1].GetId())}, tos)

	tos, err = readMapping(suite.db, getDeploymentKey(deployments[0].GetId()))
	suite.NoError(err)
	suite.Equal(SortedKeys{getImageKey(images[0].GetId()), getImageKey(images[1].GetId())}, tos)

	tos, err = readMapping(suite.db, getDeploymentKey(deployments[1].GetId()))
	suite.NoError(err)
	suite.Equal(SortedKeys(nil), tos)

	tos, err = readMapping(suite.db, getImageKey(images[0].GetId()))
	suite.NoError(err)
	suite.Equal(SortedKeys{getComponentKey(components[0].GetId()), getComponentKey(components[1].GetId()), getComponentKey(components[2].GetId())}, tos)

	tos, err = readMapping(suite.db, getImageKey(images[1].GetId()))
	suite.NoError(err)
	suite.Equal(SortedKeys{getComponentKey(components[0].GetId()), getComponentKey(components[1].GetId()), getComponentKey(components[3].GetId())}, tos)

	tos, err = readMapping(suite.db, getComponentKey(components[0].GetId()))
	suite.NoError(err)
	suite.Equal(SortedKeys(nil), tos)

	tos, err = readMapping(suite.db, getComponentKey(components[1].GetId()))
	suite.NoError(err)
	suite.Equal(SortedKeys{getCVEKey(cves[0].GetId()), getCVEKey(cves[1].GetId())}, tos)

	tos, err = readMapping(suite.db, getComponentKey(components[2].GetId()))
	suite.NoError(err)
	suite.Equal(SortedKeys{getCVEKey(cves[0].GetId()), getCVEKey(cves[1].GetId())}, tos)

	tos, err = readMapping(suite.db, getComponentKey(components[3].GetId()))
	suite.NoError(err)
	suite.Equal(SortedKeys{getCVEKey(cves[2].GetId()), getCVEKey(cves[3].GetId())}, tos)
}
