package test

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v4/pgxpool"
	clusterIndex "github.com/stackrox/rox/central/cluster/index"
	clustercveedgeIndex "github.com/stackrox/rox/central/clustercveedge/index"
	componentCVEEdgeDackbox "github.com/stackrox/rox/central/componentcveedge/dackbox"
	componentCVEEdgeIndex "github.com/stackrox/rox/central/componentcveedge/index"
	cveDackbox "github.com/stackrox/rox/central/cve/dackbox"
	cveStore "github.com/stackrox/rox/central/cve/datastore"
	imageCVEStore "github.com/stackrox/rox/central/cve/image/datastore"
	cveIndex "github.com/stackrox/rox/central/cve/index"
	cveSearch "github.com/stackrox/rox/central/cve/search"
	cveStorage "github.com/stackrox/rox/central/cve/store"
	cveDackboxStorage "github.com/stackrox/rox/central/cve/store/dackbox"
	deploymentDackbox "github.com/stackrox/rox/central/deployment/dackbox"
	deploymentStore "github.com/stackrox/rox/central/deployment/datastore"
	deploymentIndex "github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/globalindex"
	imageDackbox "github.com/stackrox/rox/central/image/dackbox"
	imageStore "github.com/stackrox/rox/central/image/datastore"
	imageIndex "github.com/stackrox/rox/central/image/index"
	componentDackbox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	componentIndex "github.com/stackrox/rox/central/imagecomponent/index"
	imageComponentEdgeDackbox "github.com/stackrox/rox/central/imagecomponentedge/dackbox"
	imageComponentEdgeIndex "github.com/stackrox/rox/central/imagecomponentedge/index"
	imageCVEEdgeDackbox "github.com/stackrox/rox/central/imagecveedge/dackbox"
	imageCVEEdgeIndex "github.com/stackrox/rox/central/imagecveedge/index"
	nodeIndex "github.com/stackrox/rox/central/node/index"
	nodecomponentedgeIndex "github.com/stackrox/rox/central/nodecomponentedge/index"
	riskDataStore "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/dackbox/indexer"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	queueMocks "github.com/stackrox/rox/pkg/dackbox/utils/queue/mocks"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

func TestCVEDataStoreSAC(t *testing.T) {
	suite.Run(t, new(cveDatastoreSACTestSuite))
}

type cveDatastoreSACTestSuite struct {
	suite.Suite

	engine *rocksdb.RocksDB
	index  bleve.Index

	pool *pgxpool.Pool

	// cluster datastore to inject data
	// deployment datastore to inject data
	deploymentStore deploymentStore.DataStore
	// node datastore to inject data
	// image datastore to inject data
	imageStore imageStore.DataStore
	// clustercveedge datastore to inject data

	// image cve datastore to read/search data
	imageCVEStore imageCVEStore.DataStore

	// node cve datastore to read/search data

	// cluster cve datastore to read/search data

	testContexts map[string]context.Context
}

func getAndConnectPostgresPool(ctx context.Context, s *suite.Suite) *pgxpool.Pool {
	src := pgtest.GetConnectionString(s.T())
	cfg, err := pgxpool.ParseConfig(src)
	s.Require().NoError(err, "failed to parse postgres config")
	pool, err := pgxpool.ConnectConfig(ctx, cfg)
	s.Require().NoError(err, "failed to connect to postgres")
	return pool
}

func getPostgresRiskStore(ctx context.Context, s *suite.Suite, pool *pgxpool.Pool, gormDB *gorm.DB) riskDataStore.DataStore {
	riskStore, err := riskDataStore.GetTestPostgresDataStore(ctx, s.T(), pool, gormDB)
	s.Require().NoError(err, "failed to instantiate risk datastore")
	return riskStore
}

func getRocksBleveRiskStore(s *suite.Suite, rocksEngine *rocksdb.RocksDB, bleveIndex bleve.Index) riskDataStore.DataStore {
	riskStore, err := riskDataStore.GetTestRocksBleveDataStore(s.T(), rocksEngine, bleveIndex)
	s.Require().NoError(err, "failed to instantiate risk store")
	return riskStore
}

func getPostgresImageStore(ctx context.Context, s *suite.Suite, pool *pgxpool.Pool, gormDB *gorm.DB, riskStore riskDataStore.DataStore) imageStore.DataStore {
	imageDataStore, err := imageStore.GetTestPostgresDataStore(ctx, s.T(), pool, gormDB, riskStore)
	s.Require().NoError(err, "failed to instantiate image store")
	return imageDataStore
}

func getRocksBleveImageStore(s *suite.Suite, dacky *dackbox.DackBox, keyFence concurrency.KeyFence, bleveIndex bleve.Index, riskStore riskDataStore.DataStore) imageStore.DataStore {
	imageDataStore, err := imageStore.GetTestRocksBleveDataStore(s.T(), dacky, keyFence, bleveIndex, riskStore)
	s.Require().NoError(err, "failed to instantiate image store")
	return imageDataStore
}

func getInitializedDackbox(rocksengine *rocksdb.RocksDB, bleveIndex bleve.Index, s *suite.Suite) *dackbox.DackBox {
	indexQ := queue.NewWaitableQueue()
	dacky, err := dackbox.NewRocksDBDackBox(rocksengine, indexQ, []byte("graph"), []byte("dirty"), []byte("valid"))
	s.Require().NoError(err, "failed to create dackbox")
	reg := indexer.NewWrapperRegistry()
	indexer.NewLazy(indexQ, reg, bleveIndex, dacky.AckIndexed).Start()
	reg.RegisterWrapper(cveDackbox.Bucket, cveIndex.Wrapper{})
	reg.RegisterWrapper(componentDackbox.Bucket, componentIndex.Wrapper{})
	reg.RegisterWrapper(componentCVEEdgeDackbox.Bucket, componentCVEEdgeIndex.Wrapper{})
	reg.RegisterWrapper(imageDackbox.Bucket, imageIndex.Wrapper{})
	reg.RegisterWrapper(imageComponentEdgeDackbox.Bucket, imageComponentEdgeIndex.Wrapper{})
	reg.RegisterWrapper(imageCVEEdgeDackbox.Bucket, imageCVEEdgeIndex.Wrapper{})
	reg.RegisterWrapper(deploymentDackbox.Bucket, deploymentIndex.Wrapper{})
	return dacky
}

func getPostgresDeploymentStore(ctx context.Context, dacky *dackbox.DackBox, keyFence concurrency.KeyFence,
	pgPool *pgxpool.Pool, gormDB *gorm.DB, bleveIndex bleve.Index, s *suite.Suite) deploymentStore.DataStore {
	deploymentDataStore, err := deploymentStore.GetTestPostgresDataStore(ctx, s.T(), pgPool, gormDB, dacky, keyFence, bleveIndex)
	s.Require().NoError(err)
	return deploymentDataStore
}

func getRocksBleveDeploymentStore(dacky *dackbox.DackBox, keyFence concurrency.KeyFence, pgPool *pgxpool.Pool,
	rocksEngine *rocksdb.RocksDB, bleveIndex bleve.Index, s *suite.Suite) deploymentStore.DataStore {
	deploymentDataStore, err := deploymentStore.GetTestRocksBleveDataStore(s.T(), rocksEngine, bleveIndex, dacky, keyFence, pgPool)
	s.Require().NoError(err)
	return deploymentDataStore
}

func getBleveCVESearcher(storage cveStorage.Store, graphProvider graph.Provider, genericCVEIndexer cveIndex.Indexer, bleveIndex bleve.Index) cveSearch.Searcher {
	clusterCVEEdgeIndexer := clustercveedgeIndex.New(bleveIndex)
	componentCVEEdgeIndexer := componentCVEEdgeIndex.New(bleveIndex)
	componentIndexer := componentIndex.New(bleveIndex)
	imageComponentEdgeIndexer := imageComponentEdgeIndex.New(bleveIndex)
	imageCVEEdgeIndexer := imageCVEEdgeIndex.New(bleveIndex)
	imageIndexer := imageIndex.New(bleveIndex)
	nodeComponentEdgeIndexer := nodecomponentedgeIndex.New(bleveIndex)
	nodeIndexer := nodeIndex.New(bleveIndex)
	deploymentIndexer := deploymentIndex.New(bleveIndex, bleveIndex)
	clusterIndexer := clusterIndex.New(bleveIndex)
	return cveSearch.New(storage, graphProvider, genericCVEIndexer, clusterCVEEdgeIndexer, componentCVEEdgeIndexer, componentIndexer, imageComponentEdgeIndexer, imageCVEEdgeIndexer, imageIndexer, nodeComponentEdgeIndexer, nodeIndexer, deploymentIndexer, clusterIndexer)
}

func (s *cveDatastoreSACTestSuite) SetupSuite() {
	var err error
	cveobj := "cveSACTest"

	mockCtrl := gomock.NewController(s.T())
	indexQ := queueMocks.NewMockWaitableQueue(mockCtrl)
	if features.PostgresDatastore.Enabled() {
		ctx := context.Background()
		s.pool = getAndConnectPostgresPool(ctx, &s.Suite)
		gormDB := pgtest.OpenGormDB(s.T(), pgtest.GetConnectionString(s.T()))
		defer pgtest.CloseGormDB(s.T(), gormDB)
		riskStore := getPostgresRiskStore(ctx, &s.Suite, s.pool, gormDB)
		s.imageStore = getPostgresImageStore(ctx, &s.Suite, s.pool, gormDB, riskStore)
		dacky := getInitializedDackbox(s.engine, s.index, &s.Suite)
		keyFence := concurrency.NewKeyFence()
		s.deploymentStore = getPostgresDeploymentStore(ctx, dacky, keyFence, s.pool, gormDB, s.index, &s.Suite)
	} else {
		// Initialize image datastore
		s.engine, err = rocksdb.NewTemp(cveobj)
		s.Require().NoError(err)
		s.index, err = globalindex.MemOnlyIndex()
		s.Require().NoError(err, "failed to create bleve index")
		dacky := getInitializedDackbox(s.engine, s.index, &s.Suite)
		keyFence := concurrency.NewKeyFence()
		riskStore := getRocksBleveRiskStore(&s.Suite, s.engine, s.index)
		s.imageStore = getRocksBleveImageStore(&s.Suite, dacky, keyFence, s.index, riskStore)
		s.deploymentStore = getRocksBleveDeploymentStore(dacky, keyFence, s.pool, s.engine, s.index, &s.Suite)
		genericCVEStorage := cveDackboxStorage.New(dacky, keyFence)
		genericCVEIndexer := cveIndex.New(s.index)
		genericCVESearcher := getBleveCVESearcher(genericCVEStorage, dacky, genericCVEIndexer, s.index)
		s.imageCVEStore, err = cveStore.New(dacky, indexQ, genericCVEStorage, genericCVEIndexer, genericCVESearcher)
		s.Require().NoError(err)
	}

	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Image)
}

func (s *cveDatastoreSACTestSuite) TearDownSuite() {
	if features.PostgresDatastore.Enabled() {
		s.pool.Close()
	} else {
		s.Require().NoError(s.index.Close())
		s.Require().NoError(rocksdb.CloseAndRemove(s.engine))
	}
}

func (s *cveDatastoreSACTestSuite) TestGetImageCVEs() {
	testImage1 := fixtures.GetPartialImageKubeProxy_1_21_5()
	testImage2 := fixtures.GetPartialImageNginX_1_14_2()
	// testDeployment1 := fixtures.GetDeploymentCoreDNS_1_8_0(uuid.NewV4().String())
	testDeployment2 := fixtures.GetScopedDeploymentNginX_1_14_2(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
	writeAllCtx := sac.WithAllAccess(context.Background())
	testCtx := s.testContexts[testutils.Cluster2NamespaceBReadWriteCtx]
	// testCVEID1 := "CVE-2011-4116"
	testCVEID2 := "CVE-2009-4487"
	var cve *storage.CVE
	var found bool
	var err error
	s.Require().NoError(s.imageStore.UpsertImage(writeAllCtx, testImage1))
	s.Require().NoError(s.imageStore.UpsertImage(writeAllCtx, testImage2))
	// s.Require().NoError(s.deploymentStore.UpsertDeployment(writeAllCtx, testDeployment1))
	s.Require().NoError(s.deploymentStore.UpsertDeployment(writeAllCtx, testDeployment2))
	// cve, found, err = s.imageCVEStore.Get(writeAllCtx, testCVEID1)
	// s.NoError(err)
	// s.True(found)
	// s.NotNil(cve)
	// cve, found, err = s.imageCVEStore.Get(testCtx, testCVEID1)
	// s.NoError(err)
	// s.False(found)
	// s.Nil(cve)
	cve, found, err = s.imageCVEStore.Get(writeAllCtx, testCVEID2)
	s.NoError(err)
	s.True(found)
	s.NotNil(cve)
	s.Equal(testCVEID2, cve.GetId())
	cve, found, err = s.imageCVEStore.Get(testCtx, testCVEID2)
	s.NoError(err)
	// s.True(found)
	// s.NotNil(cve)
	// s.Equal(testCVEID2, cve.GetId())
}
