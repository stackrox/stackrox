package globaldatastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/globalindex"
	nodeDackbox "github.com/stackrox/rox/central/node/dackbox"
	dackboxDatastore "github.com/stackrox/rox/central/node/datastore/dackbox/datastore"
	"github.com/stackrox/rox/central/node/datastore/dackbox/globaldatastore"
	nodeIndex "github.com/stackrox/rox/central/node/index"
	"github.com/stackrox/rox/central/node/index/mappings"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	dackboxConcurrency "github.com/stackrox/rox/pkg/dackbox/concurrency"
	"github.com/stackrox/rox/pkg/dackbox/indexer"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

// Following methods are not covered by unit tests: CountAllNodes, Search
// and Count. Their implementations are calling underlying functions in
// datastore. And they are covered by SAC units tests for datastore.
func TestNodeGlobalDatastoreSAC(t *testing.T) {
	suite.Run(t, new(nodeDatastoreSACSuite))
}

type nodeDatastoreSACSuite struct {
	suite.Suite

	datastore       dackboxDatastore.DataStore
	globalDatastore GlobalDataStore
	optionsMap      searchPkg.OptionsMap

	// Elements for postgres mode
	pgtestbase *pgtest.TestPostgres

	// Elements for bleve+rocksdb mode
	rocksEngine *rocksdb.RocksDB
	bleveIndex  bleve.Index
	keyFence    dackboxConcurrency.KeyFence
	indexQ      queue.WaitableQueue
	dacky       *dackbox.DackBox

	testContexts map[string]context.Context

	testNodeIDs map[string][]string
	testNodes   map[string]*storage.Node
}

func (s *nodeDatastoreSACSuite) setupPostgres() {
	var err error

	s.pgtestbase = pgtest.ForT(s.T())
	s.Require().NotNil(s.pgtestbase)
	s.datastore, err = dackboxDatastore.GetTestPostgresDataStore(s.T(), s.pgtestbase.Pool)
	s.Require().NoError(err)
	s.globalDatastore = globaldatastore.New(s.datastore)
	s.Require().NoError(err)

	s.optionsMap = schema.NodesSchema.OptionsMap
}

func (s *nodeDatastoreSACSuite) setupRocks() {
	var err error

	s.rocksEngine, err = rocksdb.NewTemp("nodeGlobalDatastoreSACTest")
	s.Require().NoError(err)
	s.bleveIndex, err = globalindex.MemOnlyIndex()
	s.Require().NoError(err)
	s.keyFence = dackboxConcurrency.NewKeyFence()
	s.indexQ = queue.NewWaitableQueue()
	s.dacky, err = dackbox.NewRocksDBDackBox(s.rocksEngine, s.indexQ, []byte("graph"), []byte("dirty"), []byte("valid"))
	s.Require().NoError(err)

	reg := indexer.NewWrapperRegistry()
	indexer.NewLazy(s.indexQ, reg, s.bleveIndex, s.dacky.AckIndexed).Start()
	reg.RegisterWrapper(nodeDackbox.Bucket, nodeIndex.Wrapper{})

	s.datastore, err = dackboxDatastore.GetTestRocksBleveDataStore(s.T(), s.rocksEngine, s.bleveIndex, s.dacky, s.keyFence)
	s.Require().NoError(err)
	s.globalDatastore = globaldatastore.New(s.datastore)
	s.Require().NoError(err)

	s.optionsMap = mappings.OptionsMap
}

func (s *nodeDatastoreSACSuite) SetupSuite() {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		s.setupPostgres()
	} else {
		s.setupRocks()
	}

	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Node)
}

func (s *nodeDatastoreSACSuite) TearDownSuite() {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		s.pgtestbase.Pool.Close()
	} else {
		s.Require().NoError(rocksdb.CloseAndRemove(s.rocksEngine))
		s.Require().NoError(s.bleveIndex.Close())
	}
}

func (s *nodeDatastoreSACSuite) SetupTest() {
	s.testNodeIDs = make(map[string][]string, 0)
	s.testNodes = make(map[string]*storage.Node, 0)

	s.initTestResourceSet()
}

func (s *nodeDatastoreSACSuite) TearDownTest() {
	for _, nodeIds := range s.testNodeIDs {
		s.Require().NoError(s.datastore.DeleteNodes(s.testContexts[testutils.UnrestrictedReadWriteCtx], nodeIds...))
	}
}

func (s *nodeDatastoreSACSuite) addTestNode(clusterID string) string {
	nodeID := uuid.NewV4().String()
	node := fixtures.GetScopedNode(nodeID, clusterID)
	node.Priority = 1

	errUpsert := s.datastore.UpsertNode(s.testContexts[testutils.UnrestrictedReadWriteCtx], node)
	s.Require().NoError(errUpsert)
	s.testNodeIDs[clusterID] = append(s.testNodeIDs[clusterID], nodeID)
	s.testNodes[nodeID] = node

	return nodeID
}

func (s *nodeDatastoreSACSuite) waitForIndexing() {
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		indexingCompleted := concurrency.NewSignal()
		s.indexQ.PushSignal(&indexingCompleted)
		<-indexingCompleted.Done()
	}
}

func (s *nodeDatastoreSACSuite) initTestResourceSet() {
	clusters := []string{testconsts.Cluster1, testconsts.Cluster2, testconsts.Cluster3}

	const numberOfNodes = 3
	for _, clusterID := range clusters {
		s.testNodeIDs[clusterID] = make([]string, 0, numberOfNodes)
		for i := 0; i < numberOfNodes; i++ {
			s.addTestNode(clusterID)
		}
	}

	s.waitForIndexing()
}

type sacMultiNodeTest struct {
	Context            context.Context
	ExpectedClusterIds []string
}

func getReadSACMultiNodeTestCases(baseContext context.Context, _ *testing.T, validClusterID string, wrongClusterID string, resources ...permissions.ResourceMetadata) map[string]sacMultiNodeTest {
	resourceHandles := make([]permissions.ResourceHandle, 0, len(resources))
	for _, r := range resources {
		resourceHandles = append(resourceHandles, r)
	}

	return map[string]sacMultiNodeTest{
		"(full) read-only can get": {
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resourceHandles...))),
			ExpectedClusterIds: []string{testconsts.Cluster1, testconsts.Cluster2, testconsts.Cluster3},
		},
		"full read-write can get": {
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resourceHandles...))),
			ExpectedClusterIds: []string{testconsts.Cluster1, testconsts.Cluster2, testconsts.Cluster3},
		},
		"full read-write on wrong cluster cannot get": {
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resourceHandles...),
					sac.ClusterScopeKeys(wrongClusterID))),
			ExpectedClusterIds: []string{},
		},
		"read-write on wrong cluster and partial namespace access cannot get": {
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resourceHandles...),
					sac.ClusterScopeKeys(wrongClusterID),
					sac.NamespaceScopeKeys("someNamespace"))),
			ExpectedClusterIds: []string{},
		},
		"read-only on right cluster can get": {
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resourceHandles...),
					sac.ClusterScopeKeys(validClusterID))),
			ExpectedClusterIds: []string{validClusterID},
		},
		"full read-write on right cluster can get": {
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resourceHandles...),
					sac.ClusterScopeKeys(validClusterID))),
			ExpectedClusterIds: []string{validClusterID},
		},
		"read-write on the right cluster and partial namespace access cannot get": {
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resourceHandles...),
					sac.ClusterScopeKeys(validClusterID),
					sac.NamespaceScopeKeys("someNamespace"))),
			ExpectedClusterIds: []string{},
		},
	}
}

func getWriteSACMultiNodeTestCases(baseContext context.Context, _ *testing.T, validClusterID string, wrongClusterID string, resources ...permissions.ResourceMetadata) map[string]sacMultiNodeTest {
	resourceHandles := make([]permissions.ResourceHandle, 0, len(resources))
	for _, r := range resources {
		resourceHandles = append(resourceHandles, r)
	}

	return map[string]sacMultiNodeTest{
		"(full) read-only cannot write": {
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resourceHandles...))),
			ExpectedClusterIds: []string{},
		},
		"full read-write can write": {
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resourceHandles...))),
			ExpectedClusterIds: []string{testconsts.Cluster1, testconsts.Cluster2, testconsts.Cluster3},
		},
		"full read-write on wrong cluster cannot write": {
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resourceHandles...),
					sac.ClusterScopeKeys(wrongClusterID))),
			ExpectedClusterIds: []string{},
		},
		"read-write on wrong cluster and partial namespace access cannot write": {
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resourceHandles...),
					sac.ClusterScopeKeys(wrongClusterID),
					sac.NamespaceScopeKeys("someNamespace"))),
			ExpectedClusterIds: []string{},
		},
		"read-only on right cluster cannot write": {
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resourceHandles...),
					sac.ClusterScopeKeys(validClusterID))),
			ExpectedClusterIds: []string{},
		},
		"full read-write on right cluster can write": {
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resourceHandles...),
					sac.ClusterScopeKeys(validClusterID))),
			ExpectedClusterIds: []string{validClusterID},
		},
		"read-write on the right cluster and partial namespace access cannot write": {
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resourceHandles...),
					sac.ClusterScopeKeys(validClusterID),
					sac.NamespaceScopeKeys("someNamespace"))),
			ExpectedClusterIds: []string{},
		},
	}
}

func (s *nodeDatastoreSACSuite) TestGetAllClusterNodeStores() {
	clusterID := testconsts.Cluster2

	cases := getReadSACMultiNodeTestCases(context.Background(), s.T(), clusterID, testconsts.WrongCluster, resources.Node)
	for name, c := range cases {
		s.Run(name, func() {
			ctx := c.Context

			results, err := s.globalDatastore.GetAllClusterNodeStores(ctx, false)
			s.NoError(err)

			fetchedClusterIDs := make([]string, 0, len(results))
			for fetchedClusterID := range results {
				fetchedClusterIDs = append(fetchedClusterIDs, fetchedClusterID)
			}
			s.ElementsMatch(c.ExpectedClusterIds, fetchedClusterIDs)
		})
	}
}

func (s *nodeDatastoreSACSuite) TestGetAllClusterNodeStoresWriteAccess() {
	clusterID := testconsts.Cluster2

	cases := getWriteSACMultiNodeTestCases(context.Background(), s.T(), clusterID, testconsts.WrongCluster, resources.Node)
	for name, c := range cases {
		s.Run(name, func() {
			ctx := c.Context

			results, err := s.globalDatastore.GetAllClusterNodeStores(ctx, true)
			s.NoError(err)

			fetchedClusterIDs := make([]string, 0, len(results))
			for fetchedClusterID := range results {
				fetchedClusterIDs = append(fetchedClusterIDs, fetchedClusterID)
			}
			s.ElementsMatch(c.ExpectedClusterIds, fetchedClusterIDs)
		})
	}
}

func (s *nodeDatastoreSACSuite) TestRemoveClusterNodeStoresSingle() {
	cases := testutils.GenericGlobalSACDeleteTestCases(s.T())
	for name, c := range cases {
		s.Run(name, func() {
			var err error

			unrestrictedCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
			ctx := s.testContexts[c.ScopeKey]

			removeClusterID := uuid.NewV4().String()
			testNodeID := s.addTestNode(removeClusterID)
			s.waitForIndexing()

			_, foundTestNode, err := s.datastore.GetNode(unrestrictedCtx, testNodeID)
			s.Require().True(foundTestNode)
			s.Require().NoError(err)

			err = s.globalDatastore.RemoveClusterNodeStores(ctx, removeClusterID)
			if c.ExpectError {
				s.ErrorIs(err, c.ExpectedError)

				// Check that node is still in datastore.
				_, foundTestNode, err = s.datastore.GetNode(unrestrictedCtx, testNodeID)
				s.True(foundTestNode)
				s.NoError(err)
			} else {
				s.NoError(err)

				// Check that node is removed from store.
				_, foundTestNode, err = s.datastore.GetNode(unrestrictedCtx, testNodeID)
				s.False(foundTestNode)
				s.NoError(err)

				// Ensure that another cluster node is available.
				_, foundTestNode, err = s.datastore.GetNode(unrestrictedCtx, s.testNodeIDs[testconsts.Cluster1][0])
				s.True(foundTestNode)
				s.NoError(err)
			}
		})
	}
}

func (s *nodeDatastoreSACSuite) TestRemoveClusterNodeStoresMulti() {
	cases := testutils.GenericGlobalSACDeleteTestCases(s.T())
	for name, c := range cases {
		s.Run(name, func() {
			var err error
			var foundTestNode bool

			unrestrictedCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
			ctx := s.testContexts[c.ScopeKey]

			var testNodeIDs []string
			var removeClusterIDs []string
			for i := 0; i < 3; i++ {
				removeClusterID := uuid.NewV4().String()
				testNodeID := s.addTestNode(removeClusterID)

				removeClusterIDs = append(removeClusterIDs, removeClusterID)
				testNodeIDs = append(testNodeIDs, testNodeID)
			}
			s.waitForIndexing()

			for _, testNodeID := range testNodeIDs {
				_, foundTestNode, err = s.datastore.GetNode(unrestrictedCtx, testNodeID)
				s.Require().True(foundTestNode)
				s.Require().NoError(err)
			}

			err = s.globalDatastore.RemoveClusterNodeStores(ctx, removeClusterIDs...)
			if c.ExpectError {
				s.ErrorIs(err, c.ExpectedError)

				// Check that nodes are still in datastore.
				for _, testNodeID := range testNodeIDs {
					_, foundTestNode, err = s.datastore.GetNode(unrestrictedCtx, testNodeID)
					s.True(foundTestNode)
					s.NoError(err)
				}
			} else {
				s.NoError(err)

				// Check that nodes are removed from store.
				for _, testNodeID := range testNodeIDs {
					_, foundTestNode, err = s.datastore.GetNode(unrestrictedCtx, testNodeID)
					s.False(foundTestNode)
					s.NoError(err)
				}

				// Ensure that another cluster node is available.
				_, foundTestNode, err = s.datastore.GetNode(unrestrictedCtx, s.testNodeIDs[testconsts.Cluster1][0])
				s.True(foundTestNode)
				s.NoError(err)
			}
		})
	}
}
