package datastore

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	edgeDataStore "github.com/stackrox/stackrox/central/clustercveedge/datastore"
	edgeIndexMocks "github.com/stackrox/stackrox/central/clustercveedge/index/mocks"
	edgeSearchMocks "github.com/stackrox/stackrox/central/clustercveedge/search/mocks"
	edgeStore "github.com/stackrox/stackrox/central/clustercveedge/store/dackbox"
	"github.com/stackrox/stackrox/central/cve/common"
	"github.com/stackrox/stackrox/central/cve/converter"
	indexMocks "github.com/stackrox/stackrox/central/cve/index/mocks"
	searchMocks "github.com/stackrox/stackrox/central/cve/search/mocks"
	store "github.com/stackrox/stackrox/central/cve/store/dackbox"
	storeMocks "github.com/stackrox/stackrox/central/cve/store/mocks"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/dackbox"
	graphMocks "github.com/stackrox/stackrox/pkg/dackbox/graph/mocks"
	"github.com/stackrox/stackrox/pkg/dackbox/utils/queue"
	queueMocks "github.com/stackrox/stackrox/pkg/dackbox/utils/queue/mocks"
	"github.com/stackrox/stackrox/pkg/sac"
	searchPkg "github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

var (
	testSuppressionQuery = searchPkg.NewQueryBuilder().AddBools(searchPkg.CVESuppressed, true).ProtoQuery()
)

func TestCVEDataStore(t *testing.T) {
	suite.Run(t, new(CVEDataStoreSuite))
}

type CVEDataStoreSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller

	indexer   *indexMocks.MockIndexer
	storage   *storeMocks.MockStore
	searcher  *searchMocks.MockSearcher
	provider  *graphMocks.MockProvider
	indexQ    *queueMocks.MockWaitableQueue
	datastore *datastoreImpl
}

func (suite *CVEDataStoreSuite) SetupSuite() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.indexer = indexMocks.NewMockIndexer(suite.mockCtrl)
	suite.storage = storeMocks.NewMockStore(suite.mockCtrl)
	suite.searcher = searchMocks.NewMockSearcher(suite.mockCtrl)
	suite.provider = graphMocks.NewMockProvider(suite.mockCtrl)
	suite.indexQ = queueMocks.NewMockWaitableQueue(suite.mockCtrl)
	suite.searcher.EXPECT().SearchRawCVEs(accessAllCtx, testSuppressionQuery).Return([]*storage.CVE{}, nil)

	ds, err := New(suite.provider, suite.indexQ, suite.storage, suite.indexer, suite.searcher)
	suite.Require().NoError(err)
	suite.datastore = ds.(*datastoreImpl)
}

func (suite *CVEDataStoreSuite) TearDownSuite() {
	suite.mockCtrl.Finish()
}

func getImageWithCVEs(cves ...string) *storage.Image {
	vulns := make([]*storage.EmbeddedVulnerability, 0, len(cves))
	for _, cve := range cves {
		vulns = append(vulns, &storage.EmbeddedVulnerability{
			Cve: cve,
		})
	}
	return &storage.Image{
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Vulns: vulns,
				},
			},
		},
	}
}

func getNodeWithCVEs(cves ...string) *storage.Node {
	vulns := make([]*storage.EmbeddedVulnerability, 0, len(cves))
	for _, cve := range cves {
		vulns = append(vulns, &storage.EmbeddedVulnerability{
			Cve: cve,
		})
	}
	return &storage.Node{
		Scan: &storage.NodeScan{
			Components: []*storage.EmbeddedNodeScanComponent{
				{
					Vulns: vulns,
				},
			},
		},
	}
}

func (suite *CVEDataStoreSuite) verifySuppressionStateImage(image *storage.Image, suppressedCVEs, unsuppressedCVEs []string) {
	cveMap := make(map[string]bool)
	for _, comp := range image.GetScan().GetComponents() {
		for _, vuln := range comp.GetVulns() {
			cveMap[vuln.Cve] = vuln.GetSuppressed()
		}
	}
	suite.verifySuppressionState(cveMap, suppressedCVEs, unsuppressedCVEs)
}

func (suite *CVEDataStoreSuite) verifySuppressionStateNode(node *storage.Node, suppressedCVEs, unsuppressedCVEs []string) {
	cveMap := make(map[string]bool)
	for _, comp := range node.GetScan().GetComponents() {
		for _, vuln := range comp.GetVulns() {
			cveMap[vuln.Cve] = vuln.GetSuppressed()
		}
	}
	suite.verifySuppressionState(cveMap, suppressedCVEs, unsuppressedCVEs)
}

func (suite *CVEDataStoreSuite) verifySuppressionState(cveMap map[string]bool, suppressedCVEs, unsuppressedCVEs []string) {
	for _, cve := range suppressedCVEs {
		val, ok := cveMap[cve]
		suite.True(ok)
		suite.True(val)
	}
	for _, cve := range unsuppressedCVEs {
		val, ok := cveMap[cve]
		suite.True(ok)
		suite.False(val)
	}
}

func (suite *CVEDataStoreSuite) TestSuppressionCacheImages() {
	// Add some results
	suite.searcher.EXPECT().SearchRawCVEs(accessAllCtx, testSuppressionQuery).Return([]*storage.CVE{
		{
			Id:         "CVE-ABC",
			Suppressed: true,
		},
		{
			Id:         "CVE-DEF",
			Suppressed: true,
		},
	}, nil)
	suite.NoError(suite.datastore.buildSuppressedCache())
	expectedCache := common.CVESuppressionCache{
		"CVE-ABC": {},
		"CVE-DEF": {},
	}
	suite.Equal(expectedCache, suite.datastore.cveSuppressionCache)

	// No apply these to the image
	img := getImageWithCVEs("CVE-ABC", "CVE-DEF", "CVE-GHI")
	suite.datastore.EnrichImageWithSuppressedCVEs(img)
	suite.verifySuppressionStateImage(img, []string{"CVE-ABC", "CVE-DEF"}, []string{"CVE-GHI"})

	start := types.TimestampNow()
	duration := types.DurationProto(10 * time.Minute)

	expiry, err := getSuppressExpiry(start, duration)
	suite.NoError(err)

	suite.storage.EXPECT().GetMany(gomock.Any(), []string{"CVE-GHI"}).Return([]*storage.CVE{{Id: "CVE-GHI"}}, nil, nil)
	storedCVE := &storage.CVE{
		Id:                 "CVE-GHI",
		Suppressed:         true,
		SuppressActivation: start,
		SuppressExpiry:     expiry,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	testAllAccessContext := sac.WithAllAccess(ctx)
	suite.storage.EXPECT().Upsert(testAllAccessContext, storedCVE).Return(nil)

	// Clear image before suppressing
	img = getImageWithCVEs("CVE-ABC", "CVE-DEF", "CVE-GHI")
	suite.indexQ.EXPECT().PushSignal(gomock.Any())
	err = suite.datastore.Suppress(testAllAccessContext, start, duration, "CVE-GHI")
	suite.Equal("timed out waiting for indexing", err.Error())
	suite.datastore.EnrichImageWithSuppressedCVEs(img)
	suite.verifySuppressionStateImage(img, []string{"CVE-ABC", "CVE-DEF", "CVE-GHI"}, nil)

	// Clear image before unsupressing
	img = getImageWithCVEs("CVE-ABC", "CVE-DEF", "CVE-GHI")
	suite.storage.EXPECT().GetMany(gomock.Any(), []string{"CVE-GHI"}).Return([]*storage.CVE{storedCVE}, nil, nil)
	suite.storage.EXPECT().Upsert(gomock.Any(), &storage.CVE{Id: "CVE-GHI"}).Return(nil)
	suite.indexQ.EXPECT().PushSignal(gomock.Any())
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	testAllAccessContext = sac.WithAllAccess(ctx)
	err = suite.datastore.Unsuppress(testAllAccessContext, "CVE-GHI")
	suite.Equal("timed out waiting for indexing", err.Error())
	suite.datastore.EnrichImageWithSuppressedCVEs(img)
	suite.verifySuppressionStateImage(img, []string{"CVE-ABC", "CVE-DEF"}, []string{"CVE-GHI"})
}

func (suite *CVEDataStoreSuite) TestSuppressionCacheNodes() {
	// Add some results
	suite.searcher.EXPECT().SearchRawCVEs(accessAllCtx, testSuppressionQuery).Return([]*storage.CVE{
		{
			Id:         "CVE-ABC",
			Suppressed: true,
		},
		{
			Id:         "CVE-DEF",
			Suppressed: true,
		},
	}, nil)
	suite.NoError(suite.datastore.buildSuppressedCache())
	expectedCache := common.CVESuppressionCache{
		"CVE-ABC": {},
		"CVE-DEF": {},
	}
	suite.Equal(expectedCache, suite.datastore.cveSuppressionCache)

	// Now apply these to the node
	node := getNodeWithCVEs("CVE-ABC", "CVE-DEF", "CVE-GHI")
	suite.datastore.EnrichNodeWithSuppressedCVEs(node)
	suite.verifySuppressionStateNode(node, []string{"CVE-ABC", "CVE-DEF"}, []string{"CVE-GHI"})

	start := types.TimestampNow()
	duration := types.DurationProto(10 * time.Minute)

	expiry, err := getSuppressExpiry(start, duration)
	suite.NoError(err)

	suite.storage.EXPECT().GetMany(gomock.Any(), []string{"CVE-GHI"}).Return([]*storage.CVE{{Id: "CVE-GHI"}}, nil, nil)
	storedCVE := &storage.CVE{
		Id:                 "CVE-GHI",
		Suppressed:         true,
		SuppressActivation: start,
		SuppressExpiry:     expiry,
	}
	suite.storage.EXPECT().Upsert(gomock.Any(), storedCVE).Return(nil)

	// Clear node before suppressing
	node = getNodeWithCVEs("CVE-ABC", "CVE-DEF", "CVE-GHI")
	suite.indexQ.EXPECT().PushSignal(gomock.Any())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	testAllAccessContext := sac.WithAllAccess(ctx)
	err = suite.datastore.Suppress(testAllAccessContext, start, duration, "CVE-GHI")
	suite.Equal("timed out waiting for indexing", err.Error())
	suite.datastore.EnrichNodeWithSuppressedCVEs(node)
	suite.verifySuppressionStateNode(node, []string{"CVE-ABC", "CVE-DEF", "CVE-GHI"}, nil)

	// Clear node before unsupressing
	node = getNodeWithCVEs("CVE-ABC", "CVE-DEF", "CVE-GHI")
	suite.storage.EXPECT().GetMany(gomock.Any(), []string{"CVE-GHI"}).Return([]*storage.CVE{storedCVE}, nil, nil)
	suite.storage.EXPECT().Upsert(gomock.Any(), &storage.CVE{Id: "CVE-GHI"}).Return(nil)
	suite.indexQ.EXPECT().PushSignal(gomock.Any())
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	testAllAccessContext = sac.WithAllAccess(ctx)
	err = suite.datastore.Unsuppress(testAllAccessContext, "CVE-GHI")
	suite.Equal("timed out waiting for indexing", err.Error())
	suite.datastore.EnrichNodeWithSuppressedCVEs(node)
	suite.verifySuppressionStateNode(node, []string{"CVE-ABC", "CVE-DEF"}, []string{"CVE-GHI"})
}

func (suite *CVEDataStoreSuite) TestMultiTypedCVEs() {
	rocksDB := rocksdbtest.RocksDBForT(suite.T())
	defer rocksdbtest.TearDownRocksDB(rocksDB)
	dacky, err := dackbox.NewRocksDBDackBox(rocksDB, nil, []byte("graph"), []byte("dirty"), []byte("valid"))
	suite.Require().NoError(err)
	suite.searcher.EXPECT().SearchRawCVEs(accessAllCtx, testSuppressionQuery).Return([]*storage.CVE{}, nil)
	datastore, err := New(dacky, queue.NewWaitableQueue(), store.New(dacky, concurrency.NewKeyFence()), suite.indexer, suite.searcher)
	suite.Require().NoError(err)
	edgeStore, err := edgeStore.New(dacky, concurrency.NewKeyFence())
	suite.Require().NoError(err)
	edgeDataStore, err := edgeDataStore.New(dacky, edgeStore, edgeIndexMocks.NewMockIndexer(suite.mockCtrl), edgeSearchMocks.NewMockSearcher(suite.mockCtrl))
	suite.Require().NoError(err)

	ctx := sac.WithAllAccess(context.Background())

	cve := &storage.CVE{
		Id:   "CVE-2021-1234",
		Type: storage.CVE_NODE_CVE,
	}
	cveClusters := []*storage.Cluster{{Id: "id"}}
	suite.NoError(edgeDataStore.Upsert(ctx, converter.NewClusterCVEParts(cve, cveClusters, "fixVersions")))

	expectedCVE := &storage.CVE{
		Id:    "CVE-2021-1234",
		Types: []storage.CVE_CVEType{storage.CVE_NODE_CVE},
	}
	storedCVE, exists, err := datastore.Get(ctx, cve.GetId())
	suite.NoError(err)
	suite.True(exists)
	suite.Equal(expectedCVE, storedCVE)

	// Add a second type for this CVE.
	cve = &storage.CVE{
		Id:   "CVE-2021-1234",
		Type: storage.CVE_IMAGE_CVE,
	}
	suite.NoError(edgeDataStore.Upsert(ctx, converter.NewClusterCVEParts(cve, cveClusters, "fixVersions")))

	expectedCVE = &storage.CVE{
		Id:    "CVE-2021-1234",
		Types: []storage.CVE_CVEType{storage.CVE_NODE_CVE, storage.CVE_IMAGE_CVE},
	}
	storedCVE, exists, err = datastore.Get(ctx, cve.GetId())
	suite.NoError(err)
	suite.True(exists)
	suite.Equal(expectedCVE, storedCVE)

	// One more time.
	cve = &storage.CVE{
		Id:   "CVE-2021-1234",
		Type: storage.CVE_K8S_CVE,
	}
	cve2 := &storage.CVE{
		Id:   "CVE-2021-1235",
		Type: storage.CVE_IMAGE_CVE,
	}
	suite.NoError(edgeDataStore.Upsert(ctx, converter.NewClusterCVEParts(cve, cveClusters, "fixVersions")))
	suite.NoError(edgeDataStore.Upsert(ctx, converter.NewClusterCVEParts(cve2, cveClusters, "fixVersions")))

	expectedCVE = &storage.CVE{
		Id:    "CVE-2021-1234",
		Types: []storage.CVE_CVEType{storage.CVE_NODE_CVE, storage.CVE_IMAGE_CVE, storage.CVE_K8S_CVE},
	}
	expectedCVE2 := &storage.CVE{
		Id:    "CVE-2021-1235",
		Types: []storage.CVE_CVEType{storage.CVE_IMAGE_CVE},
	}
	storedCVEs, err := datastore.GetBatch(ctx, []string{cve.GetId(), cve2.GetId()})
	suite.NoError(err)
	suite.Len(storedCVEs, 2)
	suite.Equal(expectedCVE, storedCVEs[0])
	suite.Equal(expectedCVE2, storedCVEs[1])

	// CVE datastore will not delete CVEs until they are no longer referenced by cluster/image/node.
	cveEdges, _ := edgeStore.GetAll()
	for _, cveEdge := range cveEdges {
		suite.NoError(edgeStore.Delete(cveEdge.GetId()))
	}
	// Delete CVE.
	suite.NoError(datastore.Delete(ctx, cve.GetId()))
	_, exists, err = datastore.Get(ctx, cve.GetId())
	suite.NoError(err)
	suite.False(exists)
}
