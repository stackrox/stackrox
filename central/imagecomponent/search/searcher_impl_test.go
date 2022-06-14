package search

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	clusterIndexer "github.com/stackrox/stackrox/central/cluster/index"
	componentCVEEdgeDackBox "github.com/stackrox/stackrox/central/componentcveedge/dackbox"
	componentCVEEdgeIndex "github.com/stackrox/stackrox/central/componentcveedge/index"
	cveDackbox "github.com/stackrox/stackrox/central/cve/dackbox"
	cveIndex "github.com/stackrox/stackrox/central/cve/index"
	deploymentIndex "github.com/stackrox/stackrox/central/deployment/index"
	"github.com/stackrox/stackrox/central/globalindex"
	imageDackBox "github.com/stackrox/stackrox/central/image/dackbox"
	imageDatastore "github.com/stackrox/stackrox/central/image/datastore"
	imageIndex "github.com/stackrox/stackrox/central/image/index"
	componentDackBox "github.com/stackrox/stackrox/central/imagecomponent/dackbox"
	componentIndex "github.com/stackrox/stackrox/central/imagecomponent/index"
	componentStore "github.com/stackrox/stackrox/central/imagecomponent/store/dackbox"
	imageComponentEdgeDackBox "github.com/stackrox/stackrox/central/imagecomponentedge/dackbox"
	imageComponentEdgeIndex "github.com/stackrox/stackrox/central/imagecomponentedge/index"
	imageCVEEdgeDackBox "github.com/stackrox/stackrox/central/imagecveedge/dackbox"
	imageCVEEdgeIndex "github.com/stackrox/stackrox/central/imagecveedge/index"
	nodeDackBox "github.com/stackrox/stackrox/central/node/dackbox"
	nodeDatastore "github.com/stackrox/stackrox/central/node/datastore/dackbox/datastore"
	nodeIndex "github.com/stackrox/stackrox/central/node/index"
	nodeComponentEdgeDackBox "github.com/stackrox/stackrox/central/nodecomponentedge/dackbox"
	nodeComponentEdgeIndex "github.com/stackrox/stackrox/central/nodecomponentedge/index"
	"github.com/stackrox/stackrox/central/ranking"
	mockRisks "github.com/stackrox/stackrox/central/risk/datastore/mocks"
	"github.com/stackrox/stackrox/central/role/resources"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/dackbox"
	"github.com/stackrox/stackrox/pkg/dackbox/indexer"
	"github.com/stackrox/stackrox/pkg/dackbox/utils/queue"
	"github.com/stackrox/stackrox/pkg/rocksdb"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/scancomponent"
	pkgSearch "github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/scoped"
	"github.com/stackrox/stackrox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

func TestImageComponentDataStore(t *testing.T) {
	suite.Run(t, new(ImageComponentSearchTestSuite))
}

type ImageComponentSearchTestSuite struct {
	suite.Suite

	db             *rocksdb.RocksDB
	blevePath      string
	indexQ         queue.WaitableQueue
	imageDataStore imageDatastore.DataStore
	nodeDataStore  nodeDatastore.DataStore
	searcher       Searcher

	mockRisk *mockRisks.MockDataStore
}

func (suite *ImageComponentSearchTestSuite) SetupSuite() {
	suite.db = rocksdbtest.RocksDBForT(suite.T())

	suite.indexQ = queue.NewWaitableQueue()

	dacky, err := dackbox.NewRocksDBDackBox(suite.db, suite.indexQ, []byte("graph"), []byte("dirty"), []byte("valid"))
	if err != nil {
		suite.FailNow("failed to create dackbox", err.Error())
	}

	suite.blevePath = suite.T().TempDir()
	blevePath := filepath.Join(suite.blevePath, "scorch.bleve")
	bleveIndex, err := globalindex.InitializeIndices("main", blevePath, globalindex.EphemeralIndex, "")
	if err != nil {
		suite.FailNow("failed to create bleve index", err.Error())
	}

	reg := indexer.NewWrapperRegistry()
	indexer.NewLazy(suite.indexQ, reg, bleveIndex, dacky.AckIndexed).Start()
	reg.RegisterWrapper(cveDackbox.Bucket, cveIndex.Wrapper{})
	reg.RegisterWrapper(componentDackBox.Bucket, componentIndex.Wrapper{})
	reg.RegisterWrapper(componentCVEEdgeDackBox.Bucket, componentCVEEdgeIndex.Wrapper{})
	reg.RegisterWrapper(imageDackBox.Bucket, imageIndex.Wrapper{})
	reg.RegisterWrapper(nodeDackBox.Bucket, nodeIndex.Wrapper{})
	reg.RegisterWrapper(imageComponentEdgeDackBox.Bucket, imageComponentEdgeIndex.Wrapper{})
	reg.RegisterWrapper(imageCVEEdgeDackBox.Bucket, imageCVEEdgeIndex.Wrapper{})
	reg.RegisterWrapper(nodeComponentEdgeDackBox.Bucket, nodeComponentEdgeIndex.Wrapper{})

	suite.mockRisk = mockRisks.NewMockDataStore(gomock.NewController(suite.T()))

	suite.imageDataStore = imageDatastore.New(dacky, concurrency.NewKeyFence(), bleveIndex, bleveIndex, false, suite.mockRisk, ranking.NewRanker(), ranking.NewRanker())
	suite.nodeDataStore = nodeDatastore.New(dacky, concurrency.NewKeyFence(), bleveIndex, suite.mockRisk, ranking.NewRanker(), ranking.NewRanker())

	index := componentIndex.New(bleveIndex)
	store, _ := componentStore.New(dacky, concurrency.NewKeyFence())
	suite.searcher = New(store, dacky, cveIndex.New(bleveIndex), componentCVEEdgeIndex.New(bleveIndex), index,
		imageComponentEdgeIndex.New(bleveIndex), imageCVEEdgeIndex.New(bleveIndex), imageIndex.New(bleveIndex),
		nodeComponentEdgeIndex.New(bleveIndex), nodeIndex.New(bleveIndex), deploymentIndex.New(bleveIndex, bleveIndex),
		clusterIndexer.New(bleveIndex))
}

func (suite *ImageComponentSearchTestSuite) TearDownSuite() {
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *ImageComponentSearchTestSuite) TestBasicSearchImage() {
	image := getTestImage("id1")

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Image),
	))

	// Sanity search.
	results, err := suite.searcher.Search(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)

	// Upsert image.
	suite.NoError(suite.imageDataStore.UpsertImage(ctx, image))

	// Ensure the CVEs are indexed.
	indexingDone := concurrency.NewSignal()
	suite.indexQ.PushSignal(&indexingDone)
	indexingDone.Wait()

	// Basic unscoped search.
	results, err = suite.searcher.Search(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 3)

	scopedCtx := scoped.Context(ctx, scoped.Scope{
		ID:    scancomponent.ComponentID("comp1", "ver1", ""),
		Level: v1.SearchCategory_IMAGE_COMPONENTS,
	})

	// Basic scoped search.
	results, err = suite.searcher.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 1)

	expectedComponent := &storage.ImageComponent{
		Id:      scancomponent.ComponentID("comp1", "ver1", ""),
		Name:    "comp1",
		Version: "ver1",
		Source:  storage.SourceType_OS,
	}

	// Search components.
	components, err := suite.searcher.SearchRawImageComponents(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.NotNil(components)
	suite.Len(components, 1)
	suite.Equal(expectedComponent, components[0])

	suite.mockRisk.EXPECT().RemoveRisk(gomock.Any(), "id1", storage.RiskSubjectType_IMAGE).Return(nil)
	suite.NoError(suite.imageDataStore.DeleteImages(ctx, image.GetId()))

	// Ensure search does not find anything.
	results, err = suite.searcher.Search(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)
}

func (suite *ImageComponentSearchTestSuite) TestBasicSearchNode() {
	node := getTestNode("id1")

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Node),
	))

	// Sanity search.
	results, err := suite.searcher.Search(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)

	// Upsert node.
	suite.NoError(suite.nodeDataStore.UpsertNode(ctx, node))

	// Ensure the CVEs are indexed.
	indexingDone := concurrency.NewSignal()
	suite.indexQ.PushSignal(&indexingDone)
	indexingDone.Wait()

	// Basic unscoped search.
	results, err = suite.searcher.Search(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 3)

	scopedCtx := scoped.Context(ctx, scoped.Scope{
		ID:    scancomponent.ComponentID("comp1", "ver1", ""),
		Level: v1.SearchCategory_IMAGE_COMPONENTS,
	})

	// Basic scoped search.
	results, err = suite.searcher.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 1)

	expectedComponent := &storage.ImageComponent{
		Id:      scancomponent.ComponentID("comp1", "ver1", ""),
		Name:    "comp1",
		Version: "ver1",
		Source:  storage.SourceType_INFRASTRUCTURE,
	}

	// Search components.
	components, err := suite.searcher.SearchRawImageComponents(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.NotNil(components)
	suite.Len(components, 1)
	suite.Equal(expectedComponent, components[0])

	suite.mockRisk.EXPECT().RemoveRisk(gomock.Any(), "id1", storage.RiskSubjectType_NODE).Return(nil)
	suite.NoError(suite.nodeDataStore.DeleteNodes(ctx, node.GetId()))

	// Ensure search does not find anything.
	results, err = suite.searcher.Search(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)
}

func getTestImage(id string) *storage.Image {
	return &storage.Image{
		Id: id,
		Scan: &storage.ImageScan{
			ScanTime: types.TimestampNow(),
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "comp1",
					Version: "ver1",
					Vulns:   []*storage.EmbeddedVulnerability{},
				},
				{
					Name:    "comp1",
					Version: "ver2",
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
		RiskScore: 30,
		Priority:  1,
	}
}

func getTestNode(id string) *storage.Node {
	return &storage.Node{
		Id: id,
		Scan: &storage.NodeScan{
			ScanTime: types.TimestampNow(),
			Components: []*storage.EmbeddedNodeScanComponent{
				{
					Name:    "comp1",
					Version: "ver1",
					Vulns:   []*storage.EmbeddedVulnerability{},
				},
				{
					Name:    "comp1",
					Version: "ver2",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:               "cve1",
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
						},
						{
							Cve:               "cve2",
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "ver3",
							},
						},
					},
				},
				{
					Name:    "comp2",
					Version: "ver1",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:               "cve1",
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "ver2",
							},
						},
						{
							Cve:               "cve2",
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
						},
					},
				},
			},
		},
		RiskScore: 30,
		Priority:  1,
	}
}
