//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	graphConfigMocks "github.com/stackrox/rox/central/networkgraph/config/datastore/mocks"
	"github.com/stackrox/rox/central/networkgraph/entity/datastore/internal/store"
	"github.com/stackrox/rox/central/networkgraph/entity/datastore/internal/store/postgres"
	dataStoreMocks "github.com/stackrox/rox/central/networkgraph/entity/datastore/mocks"
	treeMocks "github.com/stackrox/rox/central/networkgraph/entity/networktree/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/networkgraph/externalsrcs"
	"github.com/stackrox/rox/pkg/networkgraph/testutils"
	"github.com/stackrox/rox/pkg/networkgraph/tree"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	sacTestUtils "github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	sacTestCIDR = "192.168.2.0/24"

	sacTestEntityName = "cidr1"

	globalClusterID = ""
)

func TestNetworkEntityDataStoreSAC(t *testing.T) {
	suite.Run(t, new(NetworkEntityDataStoreSACTestSuite))
}

type NetworkEntityDataStoreSACTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	db          *pgtest.TestPostgres
	ds          EntityDataStore
	graphConfig *graphConfigMocks.MockDataStore
	store       store.EntityStore
	treeMgr     *treeMocks.MockManager
	dataPusher  *dataStoreMocks.MockNetworkEntityPusher

	elevatedCtx          context.Context
	noAccessCtx          context.Context
	globalReadAccessCtx  context.Context
	globalWriteAccessCtx context.Context
}

func (s *NetworkEntityDataStoreSACTestSuite) SetupSuite() {
	s.elevatedCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph)))
	s.noAccessCtx = sac.WithNoAccess(context.Background())
	s.globalReadAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph)))
	s.globalWriteAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph)))

	s.db = pgtest.ForT(s.T())
	s.store = postgres.New(s.db.DB)
}

func (s *NetworkEntityDataStoreSACTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *NetworkEntityDataStoreSACTestSuite) SetupTest() {
	ctx := sac.WithAllAccess(context.Background())
	_, err := s.db.Exec(ctx, "TRUNCATE TABLE network_entities CASCADE")
	s.Require().NoError(err)

	s.mockCtrl = gomock.NewController(s.T())
	s.graphConfig = graphConfigMocks.NewMockDataStore(s.mockCtrl)
	s.treeMgr = treeMocks.NewMockManager(s.mockCtrl)
	s.dataPusher = dataStoreMocks.NewMockNetworkEntityPusher(s.mockCtrl)

	s.treeMgr.EXPECT().Initialize(gomock.Any())
	s.ds = newEntityDataStore(s.store, s.graphConfig, s.treeMgr, s.dataPusher)
}

func getGlobalEntityID(t testing.TB) sac.ResourceID {
	entityID, err := externalsrcs.NewGlobalScopedScopedID(sacTestCIDR)
	require.NoError(t, err)
	return entityID
}

func getCluster1EntityID(t testing.TB) sac.ResourceID {
	entityID, err := externalsrcs.NewClusterScopedID(testconsts.Cluster1, sacTestCIDR)
	require.NoError(t, err)
	return entityID
}

func getCluster2EntityID(t testing.TB) sac.ResourceID {
	entityID, err := externalsrcs.NewClusterScopedID(testconsts.Cluster2, sacTestCIDR)
	require.NoError(t, err)
	return entityID
}

func getGlobalEntity(t testing.TB) *storage.NetworkEntity {
	entityID := getGlobalEntityID(t)
	return testutils.GetExtSrcNetworkEntity(
		entityID.String(),
		sacTestEntityName,
		sacTestCIDR,
		true,
		globalClusterID,
		false,
	)
}

func getCluster1Entity(t testing.TB) *storage.NetworkEntity {
	entityID := getCluster1EntityID(t)
	return testutils.GetExtSrcNetworkEntity(
		entityID.String(),
		sacTestEntityName,
		sacTestCIDR,
		true,
		testconsts.Cluster1,
		false,
	)
}

func (s *NetworkEntityDataStoreSACTestSuite) setupSACReadTest() {
	var err error

	globalEntity := getGlobalEntity(s.T())
	cluster1Entity := getCluster1Entity(s.T())

	ctx := sac.WithAllAccess(context.Background())

	s.treeMgr.EXPECT().GetNetworkTree(gomock.Any(), globalClusterID).Return(tree.NewDefaultNetworkTreeWrapper())
	err = s.ds.CreateExternalNetworkEntity(ctx, globalEntity, true)
	s.Require().NoError(err)

	var insertedCount int
	s.treeMgr.EXPECT().GetNetworkTree(gomock.Any(), testconsts.Cluster1).Return(tree.NewDefaultNetworkTreeWrapper())
	s.dataPusher.EXPECT().PushExternalNetworkEntitiesToSensor([]string{testconsts.Cluster1})
	insertedCount, err = s.ds.CreateExtNetworkEntitiesForCluster(ctx, testconsts.Cluster1, cluster1Entity)
	s.Require().NoError(err)
	s.Require().Equal(1, insertedCount)
}

type readTestCase struct {
	contextName    string
	entityID       string
	expectedFound  bool
	expectedEntity *storage.NetworkEntity
}

func getReadTestCases(t testing.TB) map[string]readTestCase {
	globalEntityID := getGlobalEntityID(t)
	cluster1EntityID := getCluster1EntityID(t)
	missingEntityID := getCluster2EntityID(t)

	globalEntity := getGlobalEntity(t)
	clusterEntity := getCluster1Entity(t)

	return map[string]readTestCase{
		"All access can get global entity": {
			contextName:    sacTestUtils.UnrestrictedReadWriteCtx,
			entityID:       globalEntityID.String(),
			expectedFound:  true,
			expectedEntity: globalEntity,
		},
		"All access can get cluster entity": {
			contextName:    sacTestUtils.UnrestrictedReadWriteCtx,
			entityID:       cluster1EntityID.String(),
			expectedFound:  true,
			expectedEntity: clusterEntity,
		},
		"All access cannot get missing entity": {
			contextName:   sacTestUtils.UnrestrictedReadWriteCtx,
			entityID:      missingEntityID.String(),
			expectedFound: false,
		},
		"Full read access can get global entity": {
			contextName:    sacTestUtils.UnrestrictedReadCtx,
			entityID:       globalEntityID.String(),
			expectedFound:  true,
			expectedEntity: globalEntity,
		},
		"Full read access can get cluster entity": {
			contextName:    sacTestUtils.UnrestrictedReadCtx,
			entityID:       cluster1EntityID.String(),
			expectedFound:  true,
			expectedEntity: clusterEntity,
		},
		"Full read access cannot get missing entity": {
			contextName:   sacTestUtils.UnrestrictedReadCtx,
			entityID:      missingEntityID.String(),
			expectedFound: false,
		},
		"Cluster full access cannot get global entity": {
			contextName:   sacTestUtils.Cluster1ReadWriteCtx,
			entityID:      globalEntityID.String(),
			expectedFound: false,
		},
		"Cluster full access can get cluster entity": {
			contextName:    sacTestUtils.Cluster1ReadWriteCtx,
			entityID:       cluster1EntityID.String(),
			expectedFound:  true,
			expectedEntity: clusterEntity,
		},
		"Cluster full access cannot get missing entity": {
			contextName:   sacTestUtils.Cluster1ReadWriteCtx,
			entityID:      missingEntityID.String(),
			expectedFound: false,
		},
		"Cluster partial access cannot get global entity": {
			contextName:   sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			entityID:      globalEntityID.String(),
			expectedFound: false,
		},
		"Cluster partial access cannot get cluster entity": {
			contextName:   sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			entityID:      cluster1EntityID.String(),
			expectedFound: false,
		},
		"Cluster partial access cannot get missing entity": {
			contextName:   sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			entityID:      missingEntityID.String(),
			expectedFound: false,
		},
		"Other cluster access cannot get global entity": {
			contextName:   sacTestUtils.Cluster2ReadWriteCtx,
			entityID:      globalEntityID.String(),
			expectedFound: false,
		},
		"Other cluster access cannot get cluster entity": {
			contextName:   sacTestUtils.Cluster2ReadWriteCtx,
			entityID:      cluster1EntityID.String(),
			expectedFound: false,
		},
		"Other cluster access cannot get missing entity": {
			contextName:   sacTestUtils.Cluster2ReadWriteCtx,
			entityID:      missingEntityID.String(),
			expectedFound: false,
		},
	}
}

type listTestCase struct {
	contextName      string
	expectedIDs      []string
	expectedEntities []*storage.NetworkEntity
}

func getListTestCases(t testing.TB) map[string]listTestCase {
	globalEntityID := getGlobalEntityID(t)
	cluster1EntityID := getCluster1EntityID(t)

	allEntityIDs := []string{globalEntityID.String(), cluster1EntityID.String()}
	clusterEntityIDs := []string{cluster1EntityID.String()}
	noEntityIDs := make([]string, 0)

	globalEntity := getGlobalEntity(t)
	cluster1Entity := getCluster1Entity(t)

	allEntities := []*storage.NetworkEntity{globalEntity, cluster1Entity}
	clusterEntities := []*storage.NetworkEntity{cluster1Entity}
	noEntities := make([]*storage.NetworkEntity, 0)

	return map[string]listTestCase{
		"All access can get global and target cluster entities": {
			contextName:      sacTestUtils.UnrestrictedReadWriteCtx,
			expectedIDs:      allEntityIDs,
			expectedEntities: allEntities,
		},
		"Full read access can get global and target cluster entities": {
			contextName:      sacTestUtils.UnrestrictedReadCtx,
			expectedIDs:      allEntityIDs,
			expectedEntities: allEntities,
		},
		"Cluster full access can only get entities for the target cluster": {
			contextName:      sacTestUtils.Cluster1ReadWriteCtx,
			expectedIDs:      clusterEntityIDs,
			expectedEntities: clusterEntities,
		},
		"Cluster partial access cannot get global nor target cluster entities": {
			contextName:      sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedIDs:      noEntityIDs,
			expectedEntities: noEntities,
		},
		"Other cluster access cannot get global nor target cluster entities": {
			contextName:      sacTestUtils.Cluster2ReadWriteCtx,
			expectedIDs:      noEntityIDs,
			expectedEntities: noEntities,
		},
	}
}

type writeTestCase struct {
	contextName   string
	expectedError error
}

func getGlobalEntityWriteTestCases(_ testing.TB) map[string]writeTestCase {
	return map[string]writeTestCase{
		"All access can create a global entity": {
			contextName:   sacTestUtils.UnrestrictedReadWriteCtx,
			expectedError: nil,
		},
		"Full read cannot create a global entity": {
			contextName:   sacTestUtils.UnrestrictedReadCtx,
			expectedError: sac.ErrResourceAccessDenied,
		},
		"Full cluster read/write cannot create a global entity": {
			contextName:   sacTestUtils.Cluster1ReadWriteCtx,
			expectedError: sac.ErrResourceAccessDenied,
		},
		"Partial cluster read/write cannot create a global entity": {
			contextName:   sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedError: sac.ErrResourceAccessDenied,
		},
	}
}

func getClusterEntityWriteTestCases(_ testing.TB) map[string]writeTestCase {
	return map[string]writeTestCase{
		"All access can create a cluster entity": {
			contextName:   sacTestUtils.UnrestrictedReadWriteCtx,
			expectedError: nil,
		},
		"Full read cannot create a cluster entity": {
			contextName:   sacTestUtils.UnrestrictedReadCtx,
			expectedError: sac.ErrResourceAccessDenied,
		},
		"Full cluster read/write can create a cluster entity for cluster in scope": {
			contextName:   sacTestUtils.Cluster1ReadWriteCtx,
			expectedError: nil,
		},
		"Partial cluster read/write cannot create a global entity": {
			contextName:   sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedError: sac.ErrResourceAccessDenied,
		},
		"Other cluster read/write cannot create a cluster entity for cluster not in scope": {
			contextName:   sacTestUtils.Cluster2ReadWriteCtx,
			expectedError: sac.ErrResourceAccessDenied,
		},
	}
}

func (s *NetworkEntityDataStoreSACTestSuite) TestExistsSAC() {
	s.setupSACReadTest()
	testContexts := sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.NetworkGraph)

	for name, tc := range getReadTestCases(s.T()) {
		s.Run(name, func() {
			ctx := testContexts[tc.contextName]
			exists, testErr := s.ds.Exists(ctx, tc.entityID)
			s.NoError(testErr)
			s.Equal(tc.expectedFound, exists)
		})
	}
}

func (s *NetworkEntityDataStoreSACTestSuite) TestGetIDsSAC() {
	s.setupSACReadTest()
	testContexts := sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.NetworkGraph)

	for name, tc := range getListTestCases(s.T()) {
		s.Run(name, func() {
			ctx := testContexts[tc.contextName]
			fetchedIDs, testErr := s.ds.GetIDs(ctx)
			s.NoError(testErr)
			s.ElementsMatch(tc.expectedIDs, fetchedIDs)
		})
	}
}

func (s *NetworkEntityDataStoreSACTestSuite) TestGetEntitySAC() {
	s.setupSACReadTest()
	testContexts := sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.NetworkGraph)

	for name, tc := range getReadTestCases(s.T()) {
		s.Run(name, func() {
			ctx := testContexts[tc.contextName]
			entity, found, testErr := s.ds.GetEntity(ctx, tc.entityID)
			s.NoError(testErr)
			if tc.expectedFound {
				s.True(found)
				protoassert.Equal(s.T(), tc.expectedEntity, entity)
			} else {
				s.False(found)
				s.Nil(entity)
			}
		})
	}
}

func (s *NetworkEntityDataStoreSACTestSuite) TestGetEntityByQuerySAC() {
	// Note: The test here reflects the observed behavior,
	// which is not the expected one from a pure scoped access control perspective
	// (entities that are not linked to the requester scope should be filtered out,
	// like global entities for users that have access scope restricted to a cluster
	// or cluster entities for clusters that are not in the access scope of the user).
	s.setupSACReadTest()
	testContexts := sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.NetworkGraph)

	globalEntityID := getGlobalEntityID(s.T())
	cluster1EntityID := getCluster1EntityID(s.T())
	missingEntityID := getCluster2EntityID(s.T())
	allEntityIDs := []string{globalEntityID.String(), cluster1EntityID.String(), missingEntityID.String()}

	globalEntity := getGlobalEntity(s.T())
	clusterEntity := getCluster1Entity(s.T())
	allEntities := []*storage.NetworkEntity{globalEntity, clusterEntity}
	// clusterEntities := []*storage.NetworkEntity{clusterEntity}
	// noEntities := []*storage.NetworkEntity{}

	for name, tc := range map[string]struct {
		contextName      string
		expectedEntities []*storage.NetworkEntity
	}{
		"All access can get global and cluster entities": {
			contextName:      sacTestUtils.UnrestrictedReadWriteCtx,
			expectedEntities: allEntities,
		},
		"Full read access can get global and cluster entities": {
			contextName:      sacTestUtils.UnrestrictedReadCtx,
			expectedEntities: allEntities,
		},
		"Cluster full access should only get entities for the target cluster but can get all": {
			contextName:      sacTestUtils.Cluster1ReadWriteCtx,
			expectedEntities: allEntities,
			// expectedEntities: clusterEntities
		},
		"Cluster partial access should not get global nor cluster entities but can": {
			contextName:      sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedEntities: allEntities,
			// expectedEntities: noEntities
		},
		"Other cluster access should not get global nor other cluster entities but can": {
			contextName:      sacTestUtils.Cluster2ReadWriteCtx,
			expectedEntities: allEntities,
			// expectedEntities: noEntities
		},
	} {
		s.Run(name, func() {
			ctx := testContexts[tc.contextName]
			query := search.NewQueryBuilder().
				AddDocIDs(allEntityIDs...).
				ProtoQuery()
			fetchedEntities, testErr := s.ds.GetEntityByQuery(ctx, query)
			s.NoError(testErr)
			protoassert.ElementsMatch(s.T(), tc.expectedEntities, fetchedEntities)
		})
	}
}

func (s *NetworkEntityDataStoreSACTestSuite) TestGetAllEntitiesForClusterSAC() {
	s.setupSACReadTest()
	testContexts := sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.NetworkGraph)

	for name, tc := range getListTestCases(s.T()) {
		s.Run(name, func() {
			ctx := testContexts[tc.contextName]
			badTargetClusterID := ""
			fetchedEntitiesFromBadCluster, badClusterTestErr := s.ds.GetAllEntitiesForCluster(ctx, badTargetClusterID)
			s.ErrorIs(badClusterTestErr, errox.InvalidArgs)
			s.Nil(fetchedEntitiesFromBadCluster)

			s.graphConfig.EXPECT().
				GetNetworkGraphConfig(gomock.Any()).
				Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: false}, nil)
			targetClusterID := testconsts.Cluster1
			fetchedEntities, testErr := s.ds.GetAllEntitiesForCluster(ctx, targetClusterID)
			s.NoError(testErr)
			protoassert.ElementsMatch(s.T(), tc.expectedEntities, fetchedEntities)
		})
	}
}

func (s *NetworkEntityDataStoreSACTestSuite) TestGetAllEntitiesSAC() {
	s.setupSACReadTest()
	testContexts := sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.NetworkGraph)

	for name, tc := range getListTestCases(s.T()) {
		s.Run(name, func() {
			ctx := testContexts[tc.contextName]
			s.graphConfig.EXPECT().
				GetNetworkGraphConfig(gomock.Any()).
				Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: false}, nil)
			fetchedEntities, testErr := s.ds.GetAllEntities(ctx)
			s.NoError(testErr)
			protoassert.ElementsMatch(s.T(), tc.expectedEntities, fetchedEntities)
		})
	}
}

func (s *NetworkEntityDataStoreSACTestSuite) TestGetAllMatchingEntitiesSAC() {
	s.setupSACReadTest()
	testContexts := sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.NetworkGraph)

	for name, tc := range getListTestCases(s.T()) {
		s.Run(name, func() {
			ctx := testContexts[tc.contextName]
			// The predicate counts the processed entities that
			// have a cluster ID in the entity scope and those
			// that do not. The goal is to check what kind of
			// access control was applied at database level.
			globalEntityCounter := 0
			clusterEntityCounter := 0
			pred := func(ne *storage.NetworkEntity) bool {
				if ne.GetScope().GetClusterId() == "" {
					globalEntityCounter++
				} else {
					clusterEntityCounter++
				}
				return true
			}
			fetchedEntities, testErr := s.ds.GetAllMatchingEntities(ctx, pred)
			s.NoError(testErr)
			protoassert.ElementsMatch(s.T(), tc.expectedEntities, fetchedEntities)
			// Walk went over all DB items (1 global entity and 1 cluster one).
			// Access control was applied after the entity was matched against the predicate.
			s.Equal(1, globalEntityCounter)
			s.Equal(1, clusterEntityCounter)
		})
	}
}

func (s *NetworkEntityDataStoreSACTestSuite) TestCreateExternalNetworkEntitySAC() {
	testContexts := sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.NetworkGraph)
	cleanupCtx := sac.WithAllAccess(context.Background())

	globalEntityID := getGlobalEntityID(s.T())
	clusterEntityID := getCluster1EntityID(s.T())

	globalEntity := getGlobalEntity(s.T())
	cluster1Entity := getCluster1Entity(s.T())

	for name, tc := range getGlobalEntityWriteTestCases(s.T()) {
		s.Run(name, func() {
			ctx := testContexts[tc.contextName]
			// for creation and removal
			s.treeMgr.EXPECT().
				GetNetworkTree(gomock.Any(), globalClusterID).
				Return(tree.NewDefaultNetworkTreeWrapper()).
				AnyTimes()
			s.dataPusher.EXPECT().
				PushExternalNetworkEntitiesToSensor([]string{""}).
				AnyTimes()
			testErr := s.ds.CreateExternalNetworkEntity(ctx, globalEntity, false)
			s.ErrorIs(testErr, tc.expectedError)
			cleanupErr := s.ds.DeleteExternalNetworkEntity(cleanupCtx, globalEntityID.String())
			s.NoError(cleanupErr)
		})
	}

	for name, tc := range getClusterEntityWriteTestCases(s.T()) {
		s.Run(name, func() {
			ctx := testContexts[tc.contextName]
			// for creation and removal
			s.treeMgr.EXPECT().
				GetNetworkTree(gomock.Any(), testconsts.Cluster1).
				Return(tree.NewDefaultNetworkTreeWrapper()).
				AnyTimes()
			s.dataPusher.EXPECT().
				PushExternalNetworkEntitiesToSensor([]string{testconsts.Cluster1}).
				AnyTimes()
			testErr := s.ds.CreateExternalNetworkEntity(ctx, cluster1Entity, false)
			s.ErrorIs(testErr, tc.expectedError)
			cleanupErr := s.ds.DeleteExternalNetworkEntity(cleanupCtx, clusterEntityID.String())
			s.NoError(cleanupErr)
		})
	}
}

func (s *NetworkEntityDataStoreSACTestSuite) TestUpdateExternalNetworkEntitySAC() {
	testContexts := sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.NetworkGraph)
	creationCtx := sac.WithAllAccess(context.Background())
	cleanupCtx := sac.WithAllAccess(context.Background())

	globalEntityID := getGlobalEntityID(s.T())
	clusterEntityID := getCluster1EntityID(s.T())

	globalEntity := getGlobalEntity(s.T())
	cluster1Entity := getCluster1Entity(s.T())
	updatedGlobalEntity := globalEntity.CloneVT()
	updatedGlobalEntity.Info.GetExternalSource().Name = "updated"
	updatedCluster1Entity := cluster1Entity.CloneVT()
	updatedCluster1Entity.Info.GetExternalSource().Name = "updated"

	for name, tc := range getGlobalEntityWriteTestCases(s.T()) {
		s.Run(name, func() {
			ctx := testContexts[tc.contextName]
			// for creation, update and removal
			s.treeMgr.EXPECT().
				GetNetworkTree(gomock.Any(), globalClusterID).
				Return(tree.NewDefaultNetworkTreeWrapper()).
				AnyTimes()
			s.dataPusher.EXPECT().
				PushExternalNetworkEntitiesToSensor([]string{""}).
				AnyTimes()
			creationErr := s.ds.CreateExternalNetworkEntity(creationCtx, globalEntity, false)
			s.NoError(creationErr)
			testErr := s.ds.UpdateExternalNetworkEntity(ctx, updatedGlobalEntity, false)
			s.ErrorIs(testErr, tc.expectedError)
			cleanupErr := s.ds.DeleteExternalNetworkEntity(cleanupCtx, globalEntityID.String())
			s.NoError(cleanupErr)
		})
	}

	for name, tc := range getClusterEntityWriteTestCases(s.T()) {
		s.Run(name, func() {
			ctx := testContexts[tc.contextName]
			// for creation, update and removal
			s.treeMgr.EXPECT().
				GetNetworkTree(gomock.Any(), testconsts.Cluster1).
				Return(tree.NewDefaultNetworkTreeWrapper()).
				AnyTimes()
			s.dataPusher.EXPECT().
				PushExternalNetworkEntitiesToSensor([]string{testconsts.Cluster1}).
				AnyTimes()
			createErr := s.ds.CreateExternalNetworkEntity(creationCtx, cluster1Entity, false)
			s.NoError(createErr)
			testErr := s.ds.UpdateExternalNetworkEntity(ctx, updatedCluster1Entity, false)
			s.ErrorIs(testErr, tc.expectedError)
			cleanupErr := s.ds.DeleteExternalNetworkEntity(cleanupCtx, clusterEntityID.String())
			s.NoError(cleanupErr)
		})
	}
}

func (s *NetworkEntityDataStoreSACTestSuite) TestCreateExtNetworkEntitiesForClusterSAC() {
	testContexts := sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.NetworkGraph)
	cleanupCtx := sac.WithAllAccess(context.Background())

	globalEntityID := getGlobalEntityID(s.T())
	clusterEntityID := getCluster1EntityID(s.T())

	globalEntity := getGlobalEntity(s.T())
	cluster1Entity := getCluster1Entity(s.T())

	for name, tc := range getGlobalEntityWriteTestCases(s.T()) {
		s.Run(name, func() {
			ctx := testContexts[tc.contextName]
			s.treeMgr.EXPECT().
				GetNetworkTree(gomock.Any(), testconsts.Cluster1).
				Return(tree.NewDefaultNetworkTreeWrapper()).
				Times(1)
			s.dataPusher.EXPECT().
				PushExternalNetworkEntitiesToSensor([]string{testconsts.Cluster1}).
				AnyTimes()
			// cleanup
			s.treeMgr.EXPECT().
				GetNetworkTree(gomock.Any(), globalClusterID).
				Return(tree.NewDefaultNetworkTreeWrapper()).
				AnyTimes()
			s.dataPusher.EXPECT().
				PushExternalNetworkEntitiesToSensor([]string{""}).
				AnyTimes()
			_, testErr := s.ds.CreateExtNetworkEntitiesForCluster(ctx, testconsts.Cluster1, globalEntity)
			if tc.expectedError != nil {
				s.Error(testErr)
			} else {
				s.NoError(testErr)
			}
			cleanupErr := s.ds.DeleteExternalNetworkEntity(cleanupCtx, globalEntityID.String())
			s.NoError(cleanupErr)
		})
	}

	for name, tc := range getClusterEntityWriteTestCases(s.T()) {
		s.Run(name, func() {
			ctx := testContexts[tc.contextName]
			s.treeMgr.EXPECT().
				GetNetworkTree(gomock.Any(), testconsts.Cluster1).
				Return(tree.NewDefaultNetworkTreeWrapper()).
				AnyTimes()
			s.dataPusher.EXPECT().
				PushExternalNetworkEntitiesToSensor([]string{""}).
				AnyTimes()
			// cleanup
			s.treeMgr.EXPECT().
				GetNetworkTree(gomock.Any(), testconsts.Cluster1).
				Return(tree.NewDefaultNetworkTreeWrapper()).
				AnyTimes()
			s.dataPusher.EXPECT().
				PushExternalNetworkEntitiesToSensor([]string{""}).
				AnyTimes()
			_, testErr := s.ds.CreateExtNetworkEntitiesForCluster(ctx, testconsts.Cluster1, cluster1Entity)
			if tc.expectedError != nil {
				s.Error(testErr)
			} else {
				s.NoError(testErr)
			}
			cleanupErr := s.ds.DeleteExternalNetworkEntity(cleanupCtx, clusterEntityID.String())
			s.NoError(cleanupErr)
		})
	}
}

func (s *NetworkEntityDataStoreSACTestSuite) TestDeleteExternalNetworkEntitySAC() {
	testContexts := sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.NetworkGraph)
	createCtx := sac.WithAllAccess(context.Background())
	cleanupCtx := sac.WithAllAccess(context.Background())

	globalEntityID := getGlobalEntityID(s.T())
	clusterEntityID := getCluster1EntityID(s.T())

	globalEntity := getGlobalEntity(s.T())
	cluster1Entity := getCluster1Entity(s.T())

	for name, tc := range getGlobalEntityWriteTestCases(s.T()) {
		s.Run(name, func() {
			ctx := testContexts[tc.contextName]
			s.treeMgr.EXPECT().
				GetNetworkTree(gomock.Any(), globalClusterID).
				Return(tree.NewDefaultNetworkTreeWrapper()).
				AnyTimes()
			s.dataPusher.EXPECT().
				PushExternalNetworkEntitiesToSensor([]string{""}).
				AnyTimes()
			creationErr := s.ds.CreateExternalNetworkEntity(createCtx, globalEntity, false)
			s.NoError(creationErr)
			testErr := s.ds.DeleteExternalNetworkEntity(ctx, globalEntityID.String())
			s.ErrorIs(testErr, tc.expectedError)
			cleanupErr := s.ds.DeleteExternalNetworkEntity(cleanupCtx, globalEntityID.String())
			s.NoError(cleanupErr)
		})
	}

	for name, tc := range getClusterEntityWriteTestCases(s.T()) {
		s.Run(name, func() {
			ctx := testContexts[tc.contextName]
			// for creation and removal
			s.treeMgr.EXPECT().
				GetNetworkTree(gomock.Any(), testconsts.Cluster1).
				Return(tree.NewDefaultNetworkTreeWrapper()).
				AnyTimes()
			s.dataPusher.EXPECT().
				PushExternalNetworkEntitiesToSensor([]string{testconsts.Cluster1}).
				AnyTimes()
			creationErr := s.ds.CreateExternalNetworkEntity(createCtx, cluster1Entity, false)
			s.NoError(creationErr)
			testErr := s.ds.DeleteExternalNetworkEntity(ctx, clusterEntityID.String())
			s.ErrorIs(testErr, tc.expectedError)
			cleanupErr := s.ds.DeleteExternalNetworkEntity(cleanupCtx, clusterEntityID.String())
			s.NoError(cleanupErr)
		})
	}
}
