package datastore

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	graphConfigMocks "github.com/stackrox/stackrox/central/networkgraph/config/datastore/mocks"
	"github.com/stackrox/stackrox/central/networkgraph/entity/datastore/internal/store"
	"github.com/stackrox/stackrox/central/networkgraph/entity/datastore/internal/store/rocksdb"
	treeMocks "github.com/stackrox/stackrox/central/networkgraph/entity/networktree/mocks"
	"github.com/stackrox/stackrox/central/role/resources"
	connMocks "github.com/stackrox/stackrox/central/sensor/service/connection/mocks"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/networkgraph/externalsrcs"
	"github.com/stackrox/stackrox/pkg/networkgraph/testutils"
	"github.com/stackrox/stackrox/pkg/networkgraph/tree"
	pkgRocksDB "github.com/stackrox/stackrox/pkg/rocksdb"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/predicate"
	"github.com/stackrox/stackrox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

var (
	cluster1 = "cluster1"
	cluster2 = "cluster2"
	trees    = map[string]tree.NetworkTree{
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

	db          *pkgRocksDB.RocksDB
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
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph)))
	suite.noAccessCtx = sac.WithNoAccess(context.Background())
	suite.globalReadAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph)))
	suite.globalWriteAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph)))

	suite.mockCtrl = gomock.NewController(suite.T())
	var err error
	suite.db, err = pkgRocksDB.NewTemp(suite.T().Name())
	if err != nil {
		suite.FailNowf("failed to create DB: %+v", err.Error())
	}
	suite.store, err = rocksdb.New(suite.db)
	if err != nil {
		suite.FailNowf("failed to create network entity store: %+v", err.Error())
	}

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.graphConfig = graphConfigMocks.NewMockDataStore(suite.mockCtrl)
	suite.treeMgr = treeMocks.NewMockManager(suite.mockCtrl)
	suite.connMgr = connMocks.NewMockManager(suite.mockCtrl)

	suite.treeMgr.EXPECT().Initialize(gomock.Any())
	suite.ds = NewEntityDataStore(suite.store, suite.graphConfig, suite.treeMgr, suite.connMgr)
}

func (suite *NetworkEntityDataStoreTestSuite) TearDownSuite() {
	suite.mockCtrl.Finish()
	rocksdbtest.TearDownRocksDB(suite.db)
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
			entity: testutils.GetExtSrcNetworkEntity(entity1ID.String(), "cidr1", "192.0.2.0/24", true, ""),
			pass:   true,
		},
		{
			// Valid entity-no name
			entity: testutils.GetExtSrcNetworkEntity(entity2ID.String(), "", "192.0.2.0/30", false, cluster1),
			pass:   true,
		},
		{
			// Invalid external source-invalid network
			entity: testutils.GetExtSrcNetworkEntity(entity3ID.String(), "cidr1", "300.0.2.0/24", false, cluster1),
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
			entity: testutils.GetExtSrcNetworkEntity(entity5ID.String(), "", "192.0.2.0/24", false, cluster2),
			pass:   true,
		},
		{
			// Invalid entity-update CIDR block
			entity:  testutils.GetExtSrcNetworkEntity(entity5ID.String(), "", "192.0.2.0/29", false, cluster2),
			pass:    false,
			skipGet: true,
		},
		{
			// Valid entity
			entity: testutils.GetExtSrcNetworkEntity(entity6ID.String(), "", "192.0.2.0/29", false, cluster2),
			pass:   true,
		},
	}

	// Test Upsert
	for _, c := range cases {
		c := c
		cluster := c.entity.GetScope().GetClusterId()
		pushSig := concurrency.NewSignal()
		if c.pass {
			suite.treeMgr.EXPECT().GetNetworkTree(gomock.Any(), cluster).Return(trees[cluster])
			if cluster == "" {
				suite.connMgr.EXPECT().PushExternalNetworkEntitiesToAllSensors(suite.elevatedCtx).DoAndReturn(
					func(ctx context.Context) error {
						pushSig.Signal()
						return nil
					})
			} else {
				suite.connMgr.EXPECT().PushExternalNetworkEntitiesToSensor(suite.elevatedCtx, cluster).DoAndReturn(
					func(ctx context.Context, clusterID string) error {
						suite.Equal(cluster, clusterID)
						pushSig.Signal()
						return nil
					})
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
		c := c
		actual, found, err := suite.ds.GetEntity(suite.globalReadAccessCtx, c.entity.GetInfo().GetId())
		if c.pass {
			suite.NoError(err)
			suite.True(found)
			suite.Equal(c.entity, actual)
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

	// Test Delete
	for _, c := range cases {
		c := c
		cluster := c.entity.GetScope().GetClusterId()
		pushSig := concurrency.NewSignal()
		if !c.pass {
			continue
		}
		suite.treeMgr.EXPECT().GetNetworkTree(gomock.Any(), cluster).Return(trees[cluster])
		if cluster == "" {
			suite.connMgr.EXPECT().PushExternalNetworkEntitiesToAllSensors(suite.elevatedCtx).DoAndReturn(
				func(ctx context.Context) error {
					pushSig.Signal()
					return nil
				})
		} else {
			suite.connMgr.EXPECT().PushExternalNetworkEntitiesToSensor(suite.elevatedCtx, cluster).DoAndReturn(
				func(ctx context.Context, clusterID string) error {
					suite.Equal(cluster, clusterID)
					pushSig.Signal()
					return nil
				})
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
		testutils.GetExtSrcNetworkEntity(entity1ID.String(), "", "192.0.2.0/30", false, cluster1),
		testutils.GetExtSrcNetworkEntity(entity2ID.String(), "", "192.0.2.0/24", false, cluster1),
		testutils.GetExtSrcNetworkEntity(entity3ID.String(), "", "192.0.2.0/29", false, cluster1),
	}

	// Batch Create
	pushSig := concurrency.NewSignal()
	suite.treeMgr.EXPECT().GetNetworkTree(gomock.Any(), cluster1).Return(trees[cluster1]).Times(3)
	suite.connMgr.EXPECT().PushExternalNetworkEntitiesToSensor(suite.elevatedCtx, cluster1).DoAndReturn(
		func(ctx context.Context, clusterID string) error {
			suite.Equal(cluster1, clusterID)
			pushSig.Signal()
			return nil
		})

	_, err = suite.ds.CreateExtNetworkEntitiesForCluster(suite.globalWriteAccessCtx, cluster1, entities...)
	suite.NoError(err)
	suite.True(concurrency.WaitWithTimeout(&pushSig, time.Second))

	// Get
	for _, entity := range entities {
		actual, found, err := suite.ds.GetEntity(suite.globalReadAccessCtx, entity.GetInfo().GetId())
		suite.NoError(err)
		suite.True(found)
		suite.Equal(entity, actual)
	}

	// Delete
	pushSig = concurrency.NewSignal()
	suite.treeMgr.EXPECT().DeleteNetworkTree(gomock.Any(), cluster1)
	suite.connMgr.EXPECT().PushExternalNetworkEntitiesToSensor(suite.elevatedCtx, cluster1).DoAndReturn(
		func(ctx context.Context, clusterID string) error {
			suite.Equal(cluster1, clusterID)
			pushSig.Signal()
			return nil
		})
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

	entity1 := testutils.GetExtSrcNetworkEntity(entity1ID.String(), "", "192.0.2.0/24", false, cluster1)
	entity2 := testutils.GetExtSrcNetworkEntity(entity2ID.String(), "", "192.0.2.0/29", false, cluster1)
	entity3 := testutils.GetExtSrcNetworkEntity(entity3ID.String(), "", "192.0.2.0/24", false, cluster2)
	entity4 := testutils.GetExtSrcNetworkEntity(entity4ID.String(), "", "192.0.2.0/29", false, cluster2)
	defaultEntity := testutils.GetExtSrcNetworkEntity(defaultEntityID.String(), "default", "192.0.2.0/30", true, "")

	cluster1ReadCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph),
			sac.ClusterScopeKeys(cluster1)))
	cluster1WriteCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph),
			sac.ClusterScopeKeys(cluster1)))
	cluster2WriteCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
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
		c := c
		cluster := c.entity.GetScope().GetClusterId()
		pushSig := concurrency.NewSignal()

		if c.pass {
			suite.treeMgr.EXPECT().GetNetworkTree(gomock.Any(), cluster).Return(trees[cluster])
			suite.connMgr.EXPECT().PushExternalNetworkEntitiesToSensor(suite.elevatedCtx, cluster).DoAndReturn(
				func(ctx context.Context, clusterID string) error {
					suite.Equal(cluster, clusterID)
					pushSig.Signal()
					return nil
				})
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
	pushSig := concurrency.NewSignal()
	suite.connMgr.EXPECT().PushExternalNetworkEntitiesToSensor(suite.elevatedCtx, cluster1).DoAndReturn(
		func(ctx context.Context, clusterID string) error {
			suite.Equal(cluster1, clusterID)
			pushSig.Signal()
			return nil
		})
	suite.ds.RegisterCluster(context.Background(), cluster1)

	suite.treeMgr.EXPECT().GetNetworkTree(gomock.Any(), cluster2).Return(trees[cluster2])
	pushSig.Reset()
	suite.connMgr.EXPECT().PushExternalNetworkEntitiesToSensor(suite.elevatedCtx, cluster2).DoAndReturn(
		func(ctx context.Context, clusterID string) error {
			suite.Equal(cluster2, clusterID)
			pushSig.Signal()
			return nil
		})
	suite.ds.RegisterCluster(context.Background(), cluster2)

	// Success-upsert default
	suite.treeMgr.EXPECT().GetNetworkTree(gomock.Any(), "").Return(trees[""])
	pushSig.Reset()
	suite.connMgr.EXPECT().PushExternalNetworkEntitiesToAllSensors(suite.elevatedCtx).DoAndReturn(
		func(ctx context.Context) error {
			pushSig.Signal()
			return nil
		})
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
	suite.ElementsMatch([]*storage.NetworkEntity{entity1, entity2}, actuals)

	// All resources accessible
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: false}, nil)
	actuals, err = suite.ds.GetAllEntities(suite.globalReadAccessCtx)
	suite.NoError(err)
	suite.Len(actuals, 5)
	suite.ElementsMatch([]*storage.NetworkEntity{entity1, entity2, entity3, entity4, defaultEntity}, actuals)

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
		c := c
		cluster := c.entity.GetScope().GetClusterId()
		pushSig := concurrency.NewSignal()

		if c.pass {
			suite.treeMgr.EXPECT().GetNetworkTree(gomock.Any(), cluster).Return(trees[cluster])
			suite.connMgr.EXPECT().PushExternalNetworkEntitiesToSensor(suite.elevatedCtx, cluster).DoAndReturn(
				func(ctx context.Context, clusterID string) error {
					suite.Equal(cluster, clusterID)
					pushSig.Signal()
					return nil
				})
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
	pushSig.Reset()
	suite.connMgr.EXPECT().PushExternalNetworkEntitiesToSensor(suite.elevatedCtx, cluster1).DoAndReturn(
		func(ctx context.Context, clusterID string) error {
			suite.Equal(cluster1, clusterID)
			pushSig.Signal()
			return nil
		})
	suite.NoError(suite.ds.DeleteExternalNetworkEntitiesForCluster(cluster1WriteCtx, cluster1))
	suite.True(concurrency.WaitWithTimeout(&pushSig, time.Second*2))
	_, found, err = suite.ds.GetEntity(suite.globalReadAccessCtx, defaultEntity.GetInfo().GetId())
	suite.NoError(err)
	suite.True(found)

	// Now deleting default entity with cluster1 permission should fail since cluster1 is removed from list.
	suite.Error(suite.ds.DeleteExternalNetworkEntity(cluster1WriteCtx, defaultEntityID.String()))

	// Success
	suite.treeMgr.EXPECT().GetNetworkTree(gomock.Any(), "").Return(trees[""])
	pushSig.Reset()
	suite.connMgr.EXPECT().PushExternalNetworkEntitiesToAllSensors(suite.elevatedCtx).DoAndReturn(
		func(ctx context.Context) error {
			pushSig.Signal()
			return nil
		})
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

	entity1 := testutils.GetExtSrcNetworkEntity(entity1ID.String(), "cidr1", "192.0.2.0/24", true, "")
	entity2 := testutils.GetExtSrcNetworkEntity(entity2ID.String(), "", "192.0.2.0/30", false, cluster1)
	entities := []*storage.NetworkEntity{entity1, entity2}

	for _, entity := range entities {
		cluster := entity.GetScope().GetClusterId()
		pushSig := concurrency.NewSignal()
		suite.treeMgr.EXPECT().GetNetworkTree(gomock.Any(), cluster).Return(trees[cluster])
		if cluster == "" {
			suite.connMgr.EXPECT().PushExternalNetworkEntitiesToAllSensors(suite.elevatedCtx).DoAndReturn(
				func(ctx context.Context) error {
					pushSig.Signal()
					return nil
				})
		} else {
			suite.connMgr.EXPECT().PushExternalNetworkEntitiesToSensor(suite.elevatedCtx, cluster).DoAndReturn(
				func(ctx context.Context, clusterID string) error {
					suite.Equal(cluster, clusterID)
					pushSig.Signal()
					return nil
				})
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
		c := c
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
		pushSig := concurrency.NewSignal()
		suite.treeMgr.EXPECT().GetNetworkTree(gomock.Any(), cluster).Return(trees[cluster])
		if cluster == "" {
			suite.connMgr.EXPECT().PushExternalNetworkEntitiesToAllSensors(suite.elevatedCtx).DoAndReturn(
				func(ctx context.Context) error {
					pushSig.Signal()
					return nil
				})
		} else {
			suite.connMgr.EXPECT().PushExternalNetworkEntitiesToSensor(suite.elevatedCtx, cluster).DoAndReturn(
				func(ctx context.Context, clusterID string) error {
					suite.Equal(cluster, clusterID)
					pushSig.Signal()
					return nil
				})
		}
		suite.NoError(suite.ds.DeleteExternalNetworkEntity(suite.globalWriteAccessCtx, entity.GetInfo().GetId()))
		suite.True(concurrency.WaitWithTimeout(&pushSig, time.Second))
	}

	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: false}, nil)
	entities, err := suite.ds.GetAllEntities(suite.globalWriteAccessCtx)
	suite.NoError(err)
	suite.Len(entities, 0)
}
