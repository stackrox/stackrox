//go:build sql_integration

package datastore

import (
	"context"
	"testing"
	"time"

	graphConfigMocks "github.com/stackrox/rox/central/networkgraph/config/datastore/mocks"
	"github.com/stackrox/rox/central/networkgraph/entity/datastore/internal/store"
	"github.com/stackrox/rox/central/networkgraph/entity/datastore/internal/store/postgres"
	dataStoreMocks "github.com/stackrox/rox/central/networkgraph/entity/datastore/mocks"
	treeMocks "github.com/stackrox/rox/central/networkgraph/entity/networktree/mocks"
	connMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
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
	"github.com/stackrox/rox/pkg/search/predicate"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	cluster1 = "cluster1"
	cluster2 = "cluster2"
)

var (
	trees = map[string]tree.NetworkTree{
		"":       tree.NewDefaultNetworkTreeWrapper(),
		cluster1: tree.NewDefaultNetworkTreeWrapper(),
		cluster2: tree.NewDefaultNetworkTreeWrapper(),
	}
)

func TestNetworkEntityDataStore(t *testing.T) {
	suite.Run(t, new(NetworkEntityDataStoreTestSuite))
}

type NetworkEntityDataStoreTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	db          *pgtest.TestPostgres
	ds          EntityDataStore
	graphConfig *graphConfigMocks.MockDataStore
	store       store.EntityStore
	treeMgr     *treeMocks.MockManager
	connMgr     *connMocks.MockManager

	elevatedCtx          context.Context
	noAccessCtx          context.Context
	globalReadAccessCtx  context.Context
	globalWriteAccessCtx context.Context
}

func (suite *NetworkEntityDataStoreTestSuite) SetupSuite() {
	suite.elevatedCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph)))
	suite.noAccessCtx = sac.WithNoAccess(context.Background())
	suite.globalReadAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph)))
	suite.globalWriteAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph)))

	suite.db = pgtest.ForT(suite.T())
	suite.store = postgres.New(suite.db.DB)
}

func (suite *NetworkEntityDataStoreTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *NetworkEntityDataStoreTestSuite) SetupTest() {
	ctx := sac.WithAllAccess(context.Background())
	_, err := suite.db.Exec(ctx, "TRUNCATE TABLE network_entities CASCADE")
	suite.Require().NoError(err)

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.graphConfig = graphConfigMocks.NewMockDataStore(suite.mockCtrl)
	suite.treeMgr = treeMocks.NewMockManager(suite.mockCtrl)
	suite.connMgr = connMocks.NewMockManager(suite.mockCtrl)

	suite.treeMgr.EXPECT().Initialize(gomock.Any())
	dataPusher := newNetworkEntityPusher(suite.connMgr)
	suite.ds = newEntityDataStore(suite.store, suite.graphConfig, suite.treeMgr, dataPusher)
}

func (suite *NetworkEntityDataStoreTestSuite) TestNetworkEntities() {
	entity1ID, err := externalsrcs.NewGlobalScopedScopedID("192.0.2.0/24")
	suite.NoError(err)
	entity2ID, err := externalsrcs.NewClusterScopedID(cluster1, "192.0.2.0/30")
	suite.NoError(err)
	entity3ID, err := externalsrcs.NewClusterScopedID(cluster1, "300.0.2.0/24")
	suite.Error(err)
	entity4ID, err := externalsrcs.NewClusterScopedID(cluster2, "192.0.2.0/24")
	suite.NoError(err)
	entity5ID, err := externalsrcs.NewClusterScopedID(cluster2, "192.0.2.0/24")
	suite.NoError(err)
	entity6ID, err := externalsrcs.NewClusterScopedID(cluster2, "192.0.2.0/29")
	suite.NoError(err)

	cases := []struct {
		entity  *storage.NetworkEntity
		pass    bool
		skipGet bool
	}{
		{
			// Valid entity
			entity: testutils.GetExtSrcNetworkEntity(entity1ID.String(), "cidr1", "192.0.2.0/24", true, "", false),
			pass:   true,
		},
		{
			// Valid entity-no name
			entity: testutils.GetExtSrcNetworkEntity(entity2ID.String(), "", "192.0.2.0/30", false, cluster1, false),
			pass:   true,
		},
		{
			// Invalid external source-invalid network
			entity: testutils.GetExtSrcNetworkEntity(entity3ID.String(), "cidr1", "300.0.2.0/24", false, cluster1, false),
			pass:   false,
		},
		{
			// Invalid external source-invalid type
			entity: &storage.NetworkEntity{
				Info: &storage.NetworkEntityInfo{
					Id:   entity4ID.String(),
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Desc: &storage.NetworkEntityInfo_ExternalSource_{
						ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
							Name: "cidr1",
							Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
								Cidr: "192.0.2.0/24",
							},
						},
					},
				},
				Scope: &storage.NetworkEntity_Scope{
					ClusterId: cluster2,
				},
			},
			pass:    false,
			skipGet: true,
		},
		{
			// Valid entity
			entity: testutils.GetExtSrcNetworkEntity(entity5ID.String(), "", "192.0.2.0/24", false, cluster2, false),
			pass:   true,
		},
		{
			// Invalid entity-update CIDR block
			entity:  testutils.GetExtSrcNetworkEntity(entity5ID.String(), "", "192.0.2.0/29", false, cluster2, false),
			pass:    false,
			skipGet: true,
		},
		{
			// Valid entity
			entity: testutils.GetExtSrcNetworkEntity(entity6ID.String(), "", "192.0.2.0/29", false, cluster2, false),
			pass:   true,
		},
	}

	// Test Upsert
	for _, c := range cases {
		cluster := c.entity.GetScope().GetClusterId()
		var pushSig concurrency.Signal
		if c.pass {
			suite.treeMgr.EXPECT().GetNetworkTree(gomock.Any(), cluster).Return(trees[cluster])
			if cluster == "" {
				pushSig = suite.expectPushExternalNetworkEntitiesToAllSensors()
			} else {
				pushSig = suite.expectPushExternalNetworkEntitiesToSensor(cluster)
			}
		}

		err := suite.ds.CreateExternalNetworkEntity(suite.globalWriteAccessCtx, c.entity, false)

		if c.pass {
			suite.NoError(err)
			suite.True(concurrency.WaitWithTimeout(&pushSig, time.Second))
		} else {
			suite.Error(err)
		}
	}

	// Test Get
	for _, c := range cases {
		if c.skipGet {
			continue
		}
		actual, found, err := suite.ds.GetEntity(suite.globalReadAccessCtx, c.entity.GetInfo().GetId())
		if c.pass {
			suite.NoError(err)
			suite.True(found)
			protoassert.Equal(suite.T(), c.entity, actual)
		} else {
			suite.False(found)
			suite.Nil(actual)
		}
	}

	// Test get matching
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: true}, nil)
	entities, err := suite.ds.GetAllEntities(suite.globalReadAccessCtx)
	suite.NoError(err)
	suite.Len(entities, 3)

	predFactory := predicate.NewFactory("test", &storage.NetworkEntity{})
	query := search.NewQueryBuilder().AddBools(search.DefaultExternalSource, false).ProtoQuery()
	pred, err := predFactory.GeneratePredicate(query)
	suite.NoError(err)
	entities, err = suite.ds.GetAllMatchingEntities(suite.globalReadAccessCtx, func(entity *storage.NetworkEntity) bool {
		return pred.Matches(entity)
	})
	suite.NoError(err)
	suite.Len(entities, 3)

	// Test get by query
	query = search.NewQueryBuilder().AddStrings(search.ExternalSourceAddress, "192.0.2.0/29").ProtoQuery()
	entities, err = suite.ds.GetEntityByQuery(suite.globalReadAccessCtx, query)
	suite.NoError(err)
	// Expect 192.0.2.0/29 and 192.0.2.0/30 - the latter is a subset of the former
	suite.Len(entities, 2)

	// Expect no matching CIDRs for this query
	query = search.NewQueryBuilder().AddStrings(search.ExternalSourceAddress, "255.255.255.0/24").ProtoQuery()
	entities, err = suite.ds.GetEntityByQuery(suite.globalReadAccessCtx, query)
	suite.NoError(err)
	suite.Len(entities, 0)

	// Test Delete
	for _, c := range cases {
		cluster := c.entity.GetScope().GetClusterId()
		if !c.pass {
			continue
		}
		suite.treeMgr.EXPECT().GetNetworkTree(gomock.Any(), cluster).Return(trees[cluster])
		var pushSig concurrency.Signal
		if cluster == "" {
			pushSig = suite.expectPushExternalNetworkEntitiesToAllSensors()
		} else {
			pushSig = suite.expectPushExternalNetworkEntitiesToSensor(cluster)
		}

		err := suite.ds.DeleteExternalNetworkEntity(suite.globalWriteAccessCtx, c.entity.GetInfo().GetId())
		suite.NoError(err)
		suite.True(concurrency.WaitWithTimeout(&pushSig, time.Second))
	}

	// Test GetAll
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: false}, nil)
	entities, err = suite.ds.GetAllEntities(suite.globalReadAccessCtx)
	suite.NoError(err)
	suite.Len(entities, 0)
}

func (suite *NetworkEntityDataStoreTestSuite) TestNetworkEntitiesBatchOps() {
	entity1ID, err := externalsrcs.NewClusterScopedID(cluster1, "192.0.2.0/30")
	suite.NoError(err)
	entity2ID, err := externalsrcs.NewClusterScopedID(cluster1, "192.0.2.0/24")
	suite.NoError(err)
	entity3ID, err := externalsrcs.NewClusterScopedID(cluster1, "192.0.2.0/29")
	suite.NoError(err)

	entities := []*storage.NetworkEntity{
		testutils.GetExtSrcNetworkEntity(entity1ID.String(), "", "192.0.2.0/30", false, cluster1, false),
		testutils.GetExtSrcNetworkEntity(entity2ID.String(), "", "192.0.2.0/24", false, cluster1, false),
		testutils.GetExtSrcNetworkEntity(entity3ID.String(), "", "192.0.2.0/29", false, cluster1, false),
	}

	// Batch Create
	suite.treeMgr.EXPECT().GetNetworkTree(gomock.Any(), cluster1).Return(trees[cluster1]).Times(3)
	pushSig := suite.expectPushExternalNetworkEntitiesToSensor(cluster1)
	_, err = suite.ds.CreateExtNetworkEntitiesForCluster(suite.globalWriteAccessCtx, cluster1, entities...)
	suite.NoError(err)
	suite.True(concurrency.WaitWithTimeout(&pushSig, time.Second))

	// Get
	for _, entity := range entities {
		actual, found, err := suite.ds.GetEntity(suite.globalReadAccessCtx, entity.GetInfo().GetId())
		suite.NoError(err)
		suite.True(found)
		protoassert.Equal(suite.T(), entity, actual)
	}

	// Delete
	suite.treeMgr.EXPECT().DeleteNetworkTree(gomock.Any(), cluster1)
	pushSig = suite.expectPushExternalNetworkEntitiesToSensor(cluster1)
	err = suite.ds.DeleteExternalNetworkEntitiesForCluster(suite.globalWriteAccessCtx, cluster1)
	suite.NoError(err)
	suite.True(concurrency.WaitWithTimeout(&pushSig, time.Second))

	// GetAll
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: false}, nil)
	entities, err = suite.ds.GetAllEntities(suite.globalReadAccessCtx)
	suite.NoError(err)
	suite.Len(entities, 0)
}

func (suite *NetworkEntityDataStoreTestSuite) TestSAC() {
	entity1ID, _ := externalsrcs.NewClusterScopedID(cluster1, "192.0.2.0/24")
	entity2ID, _ := externalsrcs.NewClusterScopedID(cluster1, "192.0.2.0/29")
	entity3ID, _ := externalsrcs.NewClusterScopedID(cluster2, "192.0.2.0/24")
	entity4ID, _ := externalsrcs.NewClusterScopedID(cluster2, "192.0.2.0/29")
	defaultEntityID, _ := externalsrcs.NewGlobalScopedScopedID("192.0.2.0/30")

	entity1 := testutils.GetExtSrcNetworkEntity(entity1ID.String(), "", "192.0.2.0/24", false, cluster1, false)
	entity2 := testutils.GetExtSrcNetworkEntity(entity2ID.String(), "", "192.0.2.0/29", false, cluster1, false)
	entity3 := testutils.GetExtSrcNetworkEntity(entity3ID.String(), "", "192.0.2.0/24", false, cluster2, false)
	entity4 := testutils.GetExtSrcNetworkEntity(entity4ID.String(), "", "192.0.2.0/29", false, cluster2, false)
	defaultEntity := testutils.GetExtSrcNetworkEntity(defaultEntityID.String(), "default", "192.0.2.0/30", true, "", false)

	cluster1ReadCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph),
			sac.ClusterScopeKeys(cluster1)))
	cluster1WriteCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph),
			sac.ClusterScopeKeys(cluster1)))
	cluster2WriteCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph),
			sac.ClusterScopeKeys(cluster2)))

	cases := []struct {
		entity *storage.NetworkEntity
		ctx    context.Context
		pass   bool
	}{
		{
			// Error-no access
			entity: entity1,
			ctx:    suite.noAccessCtx,
			pass:   false,
		},
		{
			// Error-cluster2 permissions tried to write cluster1 resource
			entity: entity1,
			ctx:    cluster2WriteCtx,
			pass:   false,
		},
		{
			// No error-all cluster access
			entity: entity1,
			ctx:    suite.globalWriteAccessCtx,
			pass:   true,
		},
		{
			// No error-cluster1 access
			entity: entity2,
			ctx:    cluster1WriteCtx,
			pass:   true,
		},
		{
			// No error-cluster2 access
			entity: entity3,
			ctx:    cluster2WriteCtx,
			pass:   true,
		},
		{
			// No error-cluster2 access
			entity: entity4,
			ctx:    suite.globalWriteAccessCtx,
			pass:   true,
		},
		{
			// Error-no access
			entity: defaultEntity,
			ctx:    suite.noAccessCtx,
			pass:   false,
		},
	}

	for _, c := range cases {
		cluster := c.entity.GetScope().GetClusterId()

		var pushSig concurrency.Signal
		if c.pass {
			suite.treeMgr.EXPECT().GetNetworkTree(gomock.Any(), cluster).Return(trees[cluster])
			pushSig = suite.expectPushExternalNetworkEntitiesToSensor(cluster)
		}

		err := suite.ds.CreateExternalNetworkEntity(c.ctx, c.entity, false)
		if c.pass {
			suite.NoError(err)
			suite.True(concurrency.WaitWithTimeout(&pushSig, time.Second*2))
		} else {
			suite.Error(err)
		}
	}

	// Register clusters to test default entity permissions.
	suite.treeMgr.EXPECT().GetNetworkTree(gomock.Any(), cluster1).Return(trees[cluster1])
	pushSig := suite.expectPushExternalNetworkEntitiesToSensor(cluster1)
	suite.ds.RegisterCluster(context.Background(), cluster1)
	suite.True(concurrency.WaitWithTimeout(&pushSig, time.Second))

	suite.treeMgr.EXPECT().GetNetworkTree(gomock.Any(), cluster2).Return(trees[cluster2])
	pushSig = suite.expectPushExternalNetworkEntitiesToSensor(cluster2)
	suite.ds.RegisterCluster(context.Background(), cluster2)
	suite.True(concurrency.WaitWithTimeout(&pushSig, time.Second))

	// Success-upsert default
	suite.treeMgr.EXPECT().GetNetworkTree(gomock.Any(), "").Return(trees[""])
	pushSig = suite.expectPushExternalNetworkEntitiesToAllSensors()
	err := suite.ds.CreateExternalNetworkEntity(suite.globalWriteAccessCtx, defaultEntity, false)
	suite.NoError(err)
	suite.True(concurrency.WaitWithTimeout(&pushSig, time.Second*2))

	// No access
	_, found, err := suite.ds.GetEntity(suite.noAccessCtx, entity1.GetInfo().GetId())
	suite.NoError(err)
	suite.False(found)

	// Success-cluster1 permissions used to read cluster1 resource
	actual, found, err := suite.ds.GetEntity(cluster1ReadCtx, entity2.GetInfo().GetId())
	suite.NoError(err)
	suite.True(found)
	suite.NotNil(actual)

	// No Access-cluster1 permissions used to read cluster2 resource
	_, found, err = suite.ds.GetEntity(cluster1ReadCtx, entity3.GetInfo().GetId())
	suite.NoError(err)
	suite.False(found)

	// Only cluster1 resources accessible
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: false}, nil)
	actuals, err := suite.ds.GetAllEntities(cluster1ReadCtx)
	suite.NoError(err)
	protoassert.ElementsMatch(suite.T(), []*storage.NetworkEntity{entity1, entity2}, actuals)

	// All resources accessible
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: false}, nil)
	actuals, err = suite.ds.GetAllEntities(suite.globalReadAccessCtx)
	suite.NoError(err)
	suite.Len(actuals, 5)
	protoassert.ElementsMatch(suite.T(), []*storage.NetworkEntity{entity1, entity2, entity3, entity4, defaultEntity}, actuals)

	// Test Deletion
	cases = []struct {
		entity *storage.NetworkEntity
		ctx    context.Context
		pass   bool
	}{
		{
			// Error-no access
			entity: entity1,
			ctx:    suite.noAccessCtx,
			pass:   false,
		},
		{
			// Error-cluster1 read-only permission
			entity: entity1,
			ctx:    cluster1ReadCtx,
			pass:   false,
		},
		{
			// No error-all cluster access
			entity: entity1,
			ctx:    suite.globalWriteAccessCtx,
			pass:   true,
		},
		{
			// Error-cluster2 write permission used for cluster1
			entity: entity2,
			ctx:    cluster2WriteCtx,
			pass:   false,
		},
		{
			// No error-cluster2 write permission used for cluster1
			entity: entity2,
			ctx:    cluster1WriteCtx,
			pass:   true,
		},
		{
			// No error-cluster2 access
			entity: entity3,
			ctx:    cluster2WriteCtx,
			pass:   true,
		},
		{
			// No error-cluster2 access
			entity: entity4,
			ctx:    suite.globalWriteAccessCtx,
			pass:   true,
		},
	}

	for _, c := range cases {
		cluster := c.entity.GetScope().GetClusterId()

		var pushSig concurrency.Signal
		if c.pass {
			suite.treeMgr.EXPECT().GetNetworkTree(gomock.Any(), cluster).Return(trees[cluster])
			pushSig = suite.expectPushExternalNetworkEntitiesToSensor(cluster)
		}

		err := suite.ds.DeleteExternalNetworkEntity(c.ctx, c.entity.GetInfo().GetId())
		if c.pass {
			suite.NoError(err)
			suite.True(concurrency.WaitWithTimeout(&pushSig, time.Second*2))
		} else {
			suite.Error(err)
		}
	}

	// Success-deleting all cluster entities skips default.
	suite.treeMgr.EXPECT().DeleteNetworkTree(gomock.Any(), cluster1)
	pushSig = suite.expectPushExternalNetworkEntitiesToSensor(cluster1)
	suite.NoError(suite.ds.DeleteExternalNetworkEntitiesForCluster(cluster1WriteCtx, cluster1))
	suite.True(concurrency.WaitWithTimeout(&pushSig, time.Second*2))
	_, found, err = suite.ds.GetEntity(suite.globalReadAccessCtx, defaultEntity.GetInfo().GetId())
	suite.NoError(err)
	suite.True(found)

	// Now deleting default entity with cluster1 permission should fail since cluster1 is removed from list.
	suite.Error(suite.ds.DeleteExternalNetworkEntity(cluster1WriteCtx, defaultEntityID.String()))

	// Success
	suite.treeMgr.EXPECT().GetNetworkTree(gomock.Any(), "").Return(trees[""])
	pushSig = suite.expectPushExternalNetworkEntitiesToAllSensors()
	suite.NoError(suite.ds.DeleteExternalNetworkEntity(suite.globalWriteAccessCtx, defaultEntityID.String()))
	suite.True(concurrency.WaitWithTimeout(&pushSig, time.Second*2))

	// Test GetAll
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: false}, nil)
	entities, err := suite.ds.GetAllEntities(suite.globalReadAccessCtx)
	suite.NoError(err)
	suite.Len(entities, 0)
}

func (suite *NetworkEntityDataStoreTestSuite) TestDefaultGraphSetting() {
	entity1ID, _ := externalsrcs.NewGlobalScopedScopedID("192.0.2.0/24")
	entity2ID, _ := externalsrcs.NewClusterScopedID(cluster1, "192.0.2.0/30")

	entity1 := testutils.GetExtSrcNetworkEntity(entity1ID.String(), "cidr1", "192.0.2.0/24", true, "", false)
	entity2 := testutils.GetExtSrcNetworkEntity(entity2ID.String(), "", "192.0.2.0/30", false, cluster1, false)
	entities := []*storage.NetworkEntity{entity1, entity2}

	for _, entity := range entities {
		cluster := entity.GetScope().GetClusterId()
		suite.treeMgr.EXPECT().GetNetworkTree(gomock.Any(), cluster).Return(trees[cluster])
		var pushSig concurrency.Signal
		if cluster == "" {
			pushSig = suite.expectPushExternalNetworkEntitiesToAllSensors()
		} else {
			pushSig = suite.expectPushExternalNetworkEntitiesToSensor(cluster)
		}
		suite.NoError(suite.ds.CreateExternalNetworkEntity(suite.globalWriteAccessCtx, entity, false))
		suite.True(concurrency.WaitWithTimeout(&pushSig, time.Second))
	}

	cases := []struct {
		graphConfig   *storage.NetworkGraphConfig
		expectedCount int
	}{
		{
			graphConfig:   &storage.NetworkGraphConfig{HideDefaultExternalSrcs: true},
			expectedCount: 1,
		},
		{
			graphConfig:   &storage.NetworkGraphConfig{HideDefaultExternalSrcs: false},
			expectedCount: 2,
		},
	}

	for _, c := range cases {
		suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(c.graphConfig, nil)
		actual, err := suite.ds.GetAllEntities(suite.globalReadAccessCtx)
		suite.NoError(err)
		suite.Len(actual, c.expectedCount)

		suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(c.graphConfig, nil)
		actual, err = suite.ds.GetAllEntitiesForCluster(suite.globalReadAccessCtx, cluster1)
		suite.NoError(err)
		suite.Len(actual, c.expectedCount)
	}

	for _, entity := range entities {
		cluster := entity.GetScope().GetClusterId()
		suite.treeMgr.EXPECT().GetNetworkTree(gomock.Any(), cluster).Return(trees[cluster])
		var pushSig concurrency.Signal
		if cluster == "" {
			pushSig = suite.expectPushExternalNetworkEntitiesToAllSensors()
		} else {
			pushSig = suite.expectPushExternalNetworkEntitiesToSensor(cluster)
		}
		suite.NoError(suite.ds.DeleteExternalNetworkEntity(suite.globalWriteAccessCtx, entity.GetInfo().GetId()))
		suite.True(concurrency.WaitWithTimeout(&pushSig, time.Second))
	}

	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: false}, nil)
	entities, err := suite.ds.GetAllEntities(suite.globalWriteAccessCtx)
	suite.NoError(err)
	suite.Len(entities, 0)
}

func (suite *NetworkEntityDataStoreTestSuite) expectPushExternalNetworkEntitiesToAllSensors() concurrency.Signal {
	signal := concurrency.NewSignal()

	suite.connMgr.EXPECT().PushExternalNetworkEntitiesToAllSensors(suite.elevatedCtx).DoAndReturn(
		func(ctx context.Context) error {
			signal.Signal()
			return nil
		})

	return signal
}

func (suite *NetworkEntityDataStoreTestSuite) expectPushExternalNetworkEntitiesToSensor(
	expectedClusterID string) concurrency.Signal {

	signal := concurrency.NewSignal()

	suite.connMgr.EXPECT().PushExternalNetworkEntitiesToSensor(suite.elevatedCtx, expectedClusterID).DoAndReturn(
		func(ctx context.Context, clusterID string) error {
			suite.Equal(expectedClusterID, clusterID)
			signal.Signal()
			return nil
		})

	return signal
}

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

func (s *NetworkEntityDataStoreSACTestSuite) setupSACReadSingleTest() ([]sac.ResourceID, []*storage.NetworkEntity) {
	var err error
	var entity1ID, entity2ID, entity3ID sac.ResourceID

	cidr := "192.168.2.0/24"
	entity1ID, err = externalsrcs.NewGlobalScopedScopedID(cidr)
	s.Require().NoError(err)
	entity2ID, err = externalsrcs.NewClusterScopedID(testconsts.Cluster1, cidr)
	s.Require().NoError(err)
	entity3ID, err = externalsrcs.NewClusterScopedID(testconsts.Cluster2, cidr)
	s.Require().NoError(err)

	entityName := "cidr1"
	global := ""
	globalEntity := testutils.GetExtSrcNetworkEntity(
		entity1ID.String(),
		entityName,
		cidr,
		true,
		global,
		false,
	)
	cluster1Entity := testutils.GetExtSrcNetworkEntity(
		entity2ID.String(),
		entityName,
		cidr,
		true,
		testconsts.Cluster1,
		false,
	)

	ctx := sac.WithAllAccess(context.Background())

	s.treeMgr.EXPECT().GetNetworkTree(gomock.Any(), global).Return(trees[""])
	err = s.ds.CreateExternalNetworkEntity(ctx, globalEntity, true)
	s.Require().NoError(err)

	var insertedCount int
	s.treeMgr.EXPECT().GetNetworkTree(gomock.Any(), testconsts.Cluster1).Return(trees[cluster1])
	s.dataPusher.EXPECT().DoPushExternalNetworkEntitiesToSensor([]string{testconsts.Cluster1})
	insertedCount, err = s.ds.CreateExtNetworkEntitiesForCluster(ctx, testconsts.Cluster1, cluster1Entity)
	s.Require().NoError(err)
	s.Require().Equal(1, insertedCount)

	return []sac.ResourceID{entity1ID, entity2ID, entity3ID}, []*storage.NetworkEntity{globalEntity, cluster1Entity}
}

func (s *NetworkEntityDataStoreSACTestSuite) TestExistsSAC() {
	entityIDs, _ := s.setupSACReadSingleTest()
	entity1ID := entityIDs[0]
	entity2ID := entityIDs[1]
	entity3ID := entityIDs[2]
	testContexts := sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.NetworkGraph)

	for name, tc := range map[string]struct {
		contextName    string
		entityID       string
		expectedExists bool
	}{
		"All access can get global entity": {
			contextName:    sacTestUtils.UnrestrictedReadWriteCtx,
			entityID:       entity1ID.String(),
			expectedExists: true,
		},
		"All access can get cluster entity": {
			contextName:    sacTestUtils.UnrestrictedReadWriteCtx,
			entityID:       entity2ID.String(),
			expectedExists: true,
		},
		"All access cannot get missing entity": {
			contextName:    sacTestUtils.UnrestrictedReadWriteCtx,
			entityID:       entity3ID.String(),
			expectedExists: false,
		},
		"Full read access can get global entity": {
			contextName:    sacTestUtils.UnrestrictedReadCtx,
			entityID:       entity1ID.String(),
			expectedExists: true,
		},
		"Full read access can get cluster entity": {
			contextName:    sacTestUtils.UnrestrictedReadCtx,
			entityID:       entity2ID.String(),
			expectedExists: true,
		},
		"Full read access cannot get missing entity": {
			contextName:    sacTestUtils.UnrestrictedReadCtx,
			entityID:       entity3ID.String(),
			expectedExists: false,
		},
		"Cluster full access cannot get global entity": {
			contextName:    sacTestUtils.Cluster1ReadWriteCtx,
			entityID:       entity1ID.String(),
			expectedExists: false,
		},
		"Cluster full access can get cluster entity": {
			contextName:    sacTestUtils.Cluster1ReadWriteCtx,
			entityID:       entity2ID.String(),
			expectedExists: true,
		},
		"Cluster full access cannot get missing entity": {
			contextName:    sacTestUtils.Cluster1ReadWriteCtx,
			entityID:       entity3ID.String(),
			expectedExists: false,
		},
		"Cluster partial access cannot get global entity": {
			contextName:    sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			entityID:       entity1ID.String(),
			expectedExists: false,
		},
		"Cluster partial access cannot get cluster entity": {
			contextName:    sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			entityID:       entity2ID.String(),
			expectedExists: false,
		},
		"Cluster partial access cannot get missing entity": {
			contextName:    sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			entityID:       entity3ID.String(),
			expectedExists: false,
		},
		"Other cluster access cannot get global entity": {
			contextName:    sacTestUtils.Cluster2ReadWriteCtx,
			entityID:       entity1ID.String(),
			expectedExists: false,
		},
		"Other cluster access cannot get cluster entity": {
			contextName:    sacTestUtils.Cluster2ReadWriteCtx,
			entityID:       entity2ID.String(),
			expectedExists: false,
		},
		"Other cluster access cannot get missing entity": {
			contextName:    sacTestUtils.Cluster2ReadWriteCtx,
			entityID:       entity3ID.String(),
			expectedExists: false,
		},
	} {
		s.Run(name, func() {
			ctx := testContexts[tc.contextName]
			exists, testErr := s.ds.Exists(ctx, tc.entityID)
			s.NoError(testErr)
			s.Equal(tc.expectedExists, exists)
		})
	}
}

func (s *NetworkEntityDataStoreSACTestSuite) TestGetIDsSAC() {
	entityIDs, _ := s.setupSACReadSingleTest()
	entity1ID := entityIDs[0]
	entity2ID := entityIDs[1]
	testContexts := sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.NetworkGraph)

	allIDs := []string{entity1ID.String(), entity2ID.String()}
	clusterIDs := []string{entity2ID.String()}
	noIDs := make([]string, 0)

	for name, tc := range map[string]struct {
		contextName string
		expectedIDs []string
	}{
		"All access can get all entity IDs": {
			contextName: sacTestUtils.UnrestrictedReadWriteCtx,
			expectedIDs: allIDs,
		},
		"Full read access can get all entity IDs": {
			contextName: sacTestUtils.UnrestrictedReadCtx,
			expectedIDs: allIDs,
		},
		"Cluster full access can only get cluster entity IDs": {
			contextName: sacTestUtils.Cluster1ReadWriteCtx,
			expectedIDs: clusterIDs,
		},
		"Cluster partial access cannot get any entity ID": {
			contextName: sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedIDs: noIDs,
		},
		"Other cluster access cannot get any entity ID": {
			contextName: sacTestUtils.Cluster2ReadWriteCtx,
			expectedIDs: noIDs,
		},
	} {
		s.Run(name, func() {
			ctx := testContexts[tc.contextName]
			fetchedIDs, testErr := s.ds.GetIDs(ctx)
			s.NoError(testErr)
			s.ElementsMatch(tc.expectedIDs, fetchedIDs)
		})
	}
}

func (s *NetworkEntityDataStoreSACTestSuite) TestGetEntitySAC() {
	entityIDs, entities := s.setupSACReadSingleTest()
	entity1ID := entityIDs[0]
	entity2ID := entityIDs[1]
	entity3ID := entityIDs[2]
	globalEntity := entities[0]
	clusterEntity := entities[1]
	testContexts := sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.NetworkGraph)

	for name, tc := range map[string]struct {
		contextName    string
		entityID       string
		expectedFound  bool
		expectedEntity *storage.NetworkEntity
	}{
		"All access can get global entity": {
			contextName:    sacTestUtils.UnrestrictedReadWriteCtx,
			entityID:       entity1ID.String(),
			expectedFound:  true,
			expectedEntity: globalEntity,
		},
		"All access can get cluster entity": {
			contextName:    sacTestUtils.UnrestrictedReadWriteCtx,
			entityID:       entity2ID.String(),
			expectedFound:  true,
			expectedEntity: clusterEntity,
		},
		"All access cannot get missing entity": {
			contextName:   sacTestUtils.UnrestrictedReadWriteCtx,
			entityID:      entity3ID.String(),
			expectedFound: false,
		},
		"Full read access can get global entity": {
			contextName:    sacTestUtils.UnrestrictedReadCtx,
			entityID:       entity1ID.String(),
			expectedFound:  true,
			expectedEntity: globalEntity,
		},
		"Full read access can get cluster entity": {
			contextName:    sacTestUtils.UnrestrictedReadCtx,
			entityID:       entity2ID.String(),
			expectedFound:  true,
			expectedEntity: clusterEntity,
		},
		"Full read access cannot get missing entity": {
			contextName:   sacTestUtils.UnrestrictedReadCtx,
			entityID:      entity3ID.String(),
			expectedFound: false,
		},
		"Cluster full access cannot get global entity": {
			contextName:   sacTestUtils.Cluster1ReadWriteCtx,
			entityID:      entity1ID.String(),
			expectedFound: false,
		},
		"Cluster full access can get cluster entity": {
			contextName:    sacTestUtils.Cluster1ReadWriteCtx,
			entityID:       entity2ID.String(),
			expectedFound:  true,
			expectedEntity: clusterEntity,
		},
		"Cluster full access cannot get missing entity": {
			contextName:   sacTestUtils.Cluster1ReadWriteCtx,
			entityID:      entity3ID.String(),
			expectedFound: false,
		},
		"Cluster partial access cannot get global entity": {
			contextName:   sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			entityID:      entity1ID.String(),
			expectedFound: false,
		},
		"Cluster partial access cannot get cluster entity": {
			contextName:   sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			entityID:      entity2ID.String(),
			expectedFound: false,
		},
		"Cluster partial access cannot get missing entity": {
			contextName:   sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			entityID:      entity3ID.String(),
			expectedFound: false,
		},
		"Other cluster access cannot get global entity": {
			contextName:   sacTestUtils.Cluster2ReadWriteCtx,
			entityID:      entity1ID.String(),
			expectedFound: false,
		},
		"Other cluster access cannot get cluster entity": {
			contextName:   sacTestUtils.Cluster2ReadWriteCtx,
			entityID:      entity2ID.String(),
			expectedFound: false,
		},
		"Other cluster access cannot get missing entity": {
			contextName:   sacTestUtils.Cluster2ReadWriteCtx,
			entityID:      entity3ID.String(),
			expectedFound: false,
		},
	} {
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
	// Note: The test here reflects the observed behaviour,
	// which is not the expected one from a pure scoped access control perspective
	// (entities that are not linked to the requester scope should be filtered out,
	// like global entities for users that have access scope restricted to a cluster
	// or cluster entities for clusters that are not in the access scope of the user).
	entityIDs, entities := s.setupSACReadSingleTest()
	entity1ID := entityIDs[0]
	entity2ID := entityIDs[1]
	entity3ID := entityIDs[2]
	globalEntity := entities[0]
	clusterEntity := entities[1]
	testContexts := sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.NetworkGraph)

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
				AddDocIDs(entity1ID.String(), entity2ID.String(), entity3ID.String()).
				ProtoQuery()
			fetchedEntities, testErr := s.ds.GetEntityByQuery(ctx, query)
			s.NoError(testErr)
			protoassert.ElementsMatch(s.T(), tc.expectedEntities, fetchedEntities)
		})
	}
}

func (s *NetworkEntityDataStoreSACTestSuite) TestGetAllEntitiesForClusterSAC() {
	_, entities := s.setupSACReadSingleTest()
	globalEntity := entities[0]
	clusterEntity := entities[1]
	testContexts := sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.NetworkGraph)

	allEntities := []*storage.NetworkEntity{globalEntity, clusterEntity}
	clusterEntities := []*storage.NetworkEntity{clusterEntity}
	noEntities := make([]*storage.NetworkEntity, 0)

	for name, tc := range map[string]struct {
		contextName      string
		expectedEntities []*storage.NetworkEntity
	}{
		"All access can get global and target cluster entities": {
			contextName:      sacTestUtils.UnrestrictedReadWriteCtx,
			expectedEntities: allEntities,
		},
		"Full read access can get gobal and target cluster entities": {
			contextName:      sacTestUtils.UnrestrictedReadCtx,
			expectedEntities: allEntities,
		},
		"Cluster full access can get entities for the target cluster": {
			contextName:      sacTestUtils.Cluster1ReadWriteCtx,
			expectedEntities: clusterEntities,
		},
		"Cluster partial access cannot get target cluster entities": {
			contextName:      sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedEntities: noEntities,
		},
		"Other cluster access cannot get target cluster entities": {
			contextName:      sacTestUtils.Cluster2ReadWriteCtx,
			expectedEntities: noEntities,
		},
	} {
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
	_, entities := s.setupSACReadSingleTest()
	globalEntity := entities[0]
	clusterEntity := entities[1]
	testContexts := sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.NetworkGraph)

	allEntities := []*storage.NetworkEntity{globalEntity, clusterEntity}
	clusterEntities := []*storage.NetworkEntity{clusterEntity}
	noEntities := make([]*storage.NetworkEntity, 0)

	for name, tc := range map[string]struct {
		contextName      string
		expectedEntities []*storage.NetworkEntity
	}{
		"All access can get global and target cluster entities": {
			contextName:      sacTestUtils.UnrestrictedReadWriteCtx,
			expectedEntities: allEntities,
		},
		"Full read access can get gobal and target cluster entities": {
			contextName:      sacTestUtils.UnrestrictedReadCtx,
			expectedEntities: allEntities,
		},
		"Cluster full access can get entities for the target cluster": {
			contextName:      sacTestUtils.Cluster1ReadWriteCtx,
			expectedEntities: clusterEntities,
		},
		"Cluster partial access cannot get target cluster entities": {
			contextName:      sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedEntities: noEntities,
		},
		"Other cluster access cannot get target cluster entities": {
			contextName:      sacTestUtils.Cluster2ReadWriteCtx,
			expectedEntities: noEntities,
		},
	} {
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
	_, entities := s.setupSACReadSingleTest()
	globalEntity := entities[0]
	clusterEntity := entities[1]
	testContexts := sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.NetworkGraph)

	allEntities := []*storage.NetworkEntity{globalEntity, clusterEntity}
	clusterEntities := []*storage.NetworkEntity{clusterEntity}
	noEntities := make([]*storage.NetworkEntity, 0)

	for name, tc := range map[string]struct {
		contextName      string
		expectedEntities []*storage.NetworkEntity
	}{
		"All access can get global and target cluster entities": {
			contextName:      sacTestUtils.UnrestrictedReadWriteCtx,
			expectedEntities: allEntities,
		},
		"Full read access can get gobal and target cluster entities": {
			contextName:      sacTestUtils.UnrestrictedReadCtx,
			expectedEntities: allEntities,
		},
		"Cluster full access can get entities for the target cluster": {
			contextName:      sacTestUtils.Cluster1ReadWriteCtx,
			expectedEntities: clusterEntities,
		},
		"Cluster partial access cannot get target cluster entities": {
			contextName:      sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedEntities: noEntities,
		},
		"Other cluster access cannot get target cluster entities": {
			contextName:      sacTestUtils.Cluster2ReadWriteCtx,
			expectedEntities: noEntities,
		},
	} {
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
			// Access control was applied after the entity match.
			s.Equal(1, globalEntityCounter)
			s.Equal(1, clusterEntityCounter)
		})
	}
}

func (s *NetworkEntityDataStoreSACTestSuite) TestCreateExternalNetworkEntitySAC() {
	testContexts := sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.NetworkGraph)
	cleanupCtx := sac.WithAllAccess(context.Background())

	var globalEntityID sac.ResourceID
	var clusterEntityID sac.ResourceID
	var err error
	cidr := "192.168.2.0/24"
	globalEntityID, err = externalsrcs.NewGlobalScopedScopedID(cidr)
	s.Require().NoError(err)
	clusterEntityID, err = externalsrcs.NewClusterScopedID(testconsts.Cluster1, cidr)
	s.Require().NoError(err)

	entityName := "cidr1"

	globalClusterID := ""
	globalEntity := testutils.GetExtSrcNetworkEntity(
		globalEntityID.String(),
		entityName,
		cidr,
		true,
		globalClusterID,
		false,
	)
	cluster1Entity := testutils.GetExtSrcNetworkEntity(
		clusterEntityID.String(),
		entityName,
		cidr,
		true,
		testconsts.Cluster1,
		false,
	)

	for name, tc := range map[string]struct {
		contextName   string
		expectedError error
	}{
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
	} {
		s.Run(name, func() {
			ctx := testContexts[tc.contextName]
			// for creation and removal
			s.treeMgr.EXPECT().
				GetNetworkTree(gomock.Any(), globalClusterID).
				Return(tree.NewDefaultNetworkTreeWrapper()).
				AnyTimes()
			s.dataPusher.EXPECT().
				DoPushExternalNetworkEntitiesToSensor([]string{""}).
				AnyTimes()
			testErr := s.ds.CreateExternalNetworkEntity(ctx, globalEntity, false)
			s.ErrorIs(testErr, tc.expectedError)
			cleanupErr := s.ds.DeleteExternalNetworkEntity(cleanupCtx, globalEntityID.String())
			s.NoError(cleanupErr)
		})
	}

	for name, tc := range map[string]struct {
		contextName   string
		expectedError error
	}{
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
	} {
		s.Run(name, func() {
			ctx := testContexts[tc.contextName]
			// for creation and removal
			s.treeMgr.EXPECT().
				GetNetworkTree(gomock.Any(), testconsts.Cluster1).
				Return(tree.NewDefaultNetworkTreeWrapper()).
				AnyTimes()
			s.dataPusher.EXPECT().
				DoPushExternalNetworkEntitiesToSensor([]string{testconsts.Cluster1}).
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

	var globalEntityID sac.ResourceID
	var clusterEntityID sac.ResourceID
	var err error
	cidr := "192.168.2.0/24"
	globalEntityID, err = externalsrcs.NewGlobalScopedScopedID(cidr)
	s.Require().NoError(err)
	clusterEntityID, err = externalsrcs.NewClusterScopedID(testconsts.Cluster1, cidr)
	s.Require().NoError(err)

	entityName := "cidr1"

	globalClusterID := ""
	globalEntity := testutils.GetExtSrcNetworkEntity(
		globalEntityID.String(),
		entityName,
		cidr,
		true,
		globalClusterID,
		false,
	)
	cluster1Entity := testutils.GetExtSrcNetworkEntity(
		clusterEntityID.String(),
		entityName,
		cidr,
		true,
		testconsts.Cluster1,
		false,
	)
	updatedGlobalEntity := globalEntity.CloneVT()
	updatedGlobalEntity.Info.GetExternalSource().Name = "updated"
	updatedCluster1Entity := cluster1Entity.CloneVT()
	updatedCluster1Entity.Info.GetExternalSource().Name = "updated"

	for name, tc := range map[string]struct {
		contextName   string
		expectedError error
	}{
		"All access can update a global entity": {
			contextName:   sacTestUtils.UnrestrictedReadWriteCtx,
			expectedError: nil,
		},
		"Full read cannot update a global entity": {
			contextName:   sacTestUtils.UnrestrictedReadCtx,
			expectedError: sac.ErrResourceAccessDenied,
		},
		"Full cluster read/write cannot update a global entity": {
			contextName:   sacTestUtils.Cluster1ReadWriteCtx,
			expectedError: sac.ErrResourceAccessDenied,
		},
		"Partial cluster read/write cannot update a global entity": {
			contextName:   sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedError: sac.ErrResourceAccessDenied,
		},
	} {
		s.Run(name, func() {
			ctx := testContexts[tc.contextName]
			// for creation, update and removal
			s.treeMgr.EXPECT().
				GetNetworkTree(gomock.Any(), globalClusterID).
				Return(tree.NewDefaultNetworkTreeWrapper()).
				AnyTimes()
			s.dataPusher.EXPECT().
				DoPushExternalNetworkEntitiesToSensor([]string{""}).
				AnyTimes()
			creationErr := s.ds.CreateExternalNetworkEntity(creationCtx, globalEntity, false)
			s.NoError(creationErr)
			testErr := s.ds.UpdateExternalNetworkEntity(ctx, updatedGlobalEntity, false)
			s.ErrorIs(testErr, tc.expectedError)
			cleanupErr := s.ds.DeleteExternalNetworkEntity(cleanupCtx, globalEntityID.String())
			s.NoError(cleanupErr)
		})
	}

	for name, tc := range map[string]struct {
		contextName   string
		expectedError error
	}{
		"All access can update a cluster entity": {
			contextName:   sacTestUtils.UnrestrictedReadWriteCtx,
			expectedError: nil,
		},
		"Full read cannot update a cluster entity": {
			contextName:   sacTestUtils.UnrestrictedReadCtx,
			expectedError: sac.ErrResourceAccessDenied,
		},
		"Full cluster read/write can update a cluster entity for cluster in scope": {
			contextName:   sacTestUtils.Cluster1ReadWriteCtx,
			expectedError: nil,
		},
		"Partial cluster read/write cannot update a global entity": {
			contextName:   sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedError: sac.ErrResourceAccessDenied,
		},
		"Other cluster read/write cannot update a cluster entity for cluster not in scope": {
			contextName:   sacTestUtils.Cluster2ReadWriteCtx,
			expectedError: sac.ErrResourceAccessDenied,
		},
	} {
		s.Run(name, func() {
			ctx := testContexts[tc.contextName]
			// for creation, update and removal
			s.treeMgr.EXPECT().
				GetNetworkTree(gomock.Any(), testconsts.Cluster1).
				Return(tree.NewDefaultNetworkTreeWrapper()).
				AnyTimes()
			s.dataPusher.EXPECT().
				DoPushExternalNetworkEntitiesToSensor([]string{testconsts.Cluster1}).
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

	var globalEntityID sac.ResourceID
	var clusterEntityID sac.ResourceID
	var err error
	cidr := "192.168.2.0/24"
	globalEntityID, err = externalsrcs.NewGlobalScopedScopedID(cidr)
	s.Require().NoError(err)
	clusterEntityID, err = externalsrcs.NewClusterScopedID(testconsts.Cluster1, cidr)
	s.Require().NoError(err)

	entityName := "cidr1"

	globalClusterID := ""
	globalEntity := testutils.GetExtSrcNetworkEntity(
		globalEntityID.String(),
		entityName,
		cidr,
		true,
		globalClusterID,
		false,
	)
	cluster1Entity := testutils.GetExtSrcNetworkEntity(
		clusterEntityID.String(),
		entityName,
		cidr,
		true,
		testconsts.Cluster1,
		false,
	)

	for name, tc := range map[string]struct {
		contextName   string
		expectedError error
	}{
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
	} {
		s.Run(name, func() {
			ctx := testContexts[tc.contextName]
			s.treeMgr.EXPECT().
				GetNetworkTree(gomock.Any(), testconsts.Cluster1).
				Return(tree.NewDefaultNetworkTreeWrapper()).
				Times(1)
			s.dataPusher.EXPECT().
				DoPushExternalNetworkEntitiesToSensor([]string{testconsts.Cluster1}).
				AnyTimes()
			// cleanup
			s.treeMgr.EXPECT().
				GetNetworkTree(gomock.Any(), globalClusterID).
				Return(tree.NewDefaultNetworkTreeWrapper()).
				AnyTimes()
			s.dataPusher.EXPECT().
				DoPushExternalNetworkEntitiesToSensor([]string{""}).
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

	for name, tc := range map[string]struct {
		contextName   string
		expectedError error
	}{
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
	} {
		s.Run(name, func() {
			ctx := testContexts[tc.contextName]
			s.treeMgr.EXPECT().
				GetNetworkTree(gomock.Any(), testconsts.Cluster1).
				Return(tree.NewDefaultNetworkTreeWrapper()).
				AnyTimes()
			s.dataPusher.EXPECT().
				DoPushExternalNetworkEntitiesToSensor([]string{""}).
				AnyTimes()
			// cleanup
			s.treeMgr.EXPECT().
				GetNetworkTree(gomock.Any(), testconsts.Cluster1).
				Return(tree.NewDefaultNetworkTreeWrapper()).
				AnyTimes()
			s.dataPusher.EXPECT().
				DoPushExternalNetworkEntitiesToSensor([]string{""}).
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

	var globalEntityID sac.ResourceID
	var clusterEntityID sac.ResourceID
	var err error
	cidr := "192.168.2.0/24"
	globalEntityID, err = externalsrcs.NewGlobalScopedScopedID(cidr)
	s.Require().NoError(err)
	clusterEntityID, err = externalsrcs.NewClusterScopedID(testconsts.Cluster1, cidr)
	s.Require().NoError(err)

	entityName := "cidr1"

	globalClusterID := ""
	globalEntity := testutils.GetExtSrcNetworkEntity(
		globalEntityID.String(),
		entityName,
		cidr,
		true,
		globalClusterID,
		false,
	)
	cluster1Entity := testutils.GetExtSrcNetworkEntity(
		clusterEntityID.String(),
		entityName,
		cidr,
		true,
		testconsts.Cluster1,
		false,
	)

	for name, tc := range map[string]struct {
		contextName   string
		expectedError error
	}{
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
	} {
		s.Run(name, func() {
			ctx := testContexts[tc.contextName]
			s.treeMgr.EXPECT().
				GetNetworkTree(gomock.Any(), globalClusterID).
				Return(tree.NewDefaultNetworkTreeWrapper()).
				AnyTimes()
			s.dataPusher.EXPECT().
				DoPushExternalNetworkEntitiesToSensor([]string{""}).
				AnyTimes()
			creationErr := s.ds.CreateExternalNetworkEntity(createCtx, globalEntity, false)
			s.NoError(creationErr)
			testErr := s.ds.DeleteExternalNetworkEntity(ctx, globalEntityID.String())
			s.ErrorIs(testErr, tc.expectedError)
			cleanupErr := s.ds.DeleteExternalNetworkEntity(cleanupCtx, globalEntityID.String())
			s.NoError(cleanupErr)
		})
	}

	for name, tc := range map[string]struct {
		contextName   string
		expectedError error
	}{
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
	} {
		s.Run(name, func() {
			ctx := testContexts[tc.contextName]
			// for creation and removal
			s.treeMgr.EXPECT().
				GetNetworkTree(gomock.Any(), testconsts.Cluster1).
				Return(tree.NewDefaultNetworkTreeWrapper()).
				AnyTimes()
			s.dataPusher.EXPECT().
				DoPushExternalNetworkEntitiesToSensor([]string{testconsts.Cluster1}).
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
