package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	graphConfigMocks "github.com/stackrox/rox/central/networkgraph/config/datastore/mocks"
	"github.com/stackrox/rox/central/networkgraph/entity/datastore/internal/store"
	"github.com/stackrox/rox/central/networkgraph/entity/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	pkgRocksDB "github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
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
}

func (suite *NetworkEntityDataStoreTestSuite) SetupSuite() {
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
	suite.ds = NewEntityDataStore(suite.store, suite.graphConfig)
}

func (suite *NetworkEntityDataStoreTestSuite) TearDownSuite() {
	suite.mockCtrl.Finish()
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *NetworkEntityDataStoreTestSuite) TestNetworkEntities() {
	ctx := sac.WithAllAccess(context.Background())
	cluster1 := "cluster1"
	cluster2 := "cluster2"

	entity1ID, _ := sac.NewClusterScopeResourceID(cluster1, uuid.NewV4().String())
	entity2ID, _ := sac.NewClusterScopeResourceID(cluster1, uuid.NewV4().String())
	entity3ID, _ := sac.NewClusterScopeResourceID(cluster1, uuid.NewV4().String())
	entity4ID, _ := sac.NewClusterScopeResourceID(cluster2, uuid.NewV4().String())
	entity5ID, _ := sac.NewClusterScopeResourceID(cluster2, uuid.NewV4().String())
	entity6ID, _ := sac.NewClusterScopeResourceID(cluster2, uuid.NewV4().String())
	entity7ID, _ := sac.NewClusterScopeResourceID(cluster2, uuid.NewV4().String())

	// Test Add
	// Valid entity
	entity1 := &storage.NetworkEntity{
		Info: &storage.NetworkEntityInfo{
			Id:   entity1ID.String(),
			Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
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
			ClusterId: cluster1,
		},
	}
	err := suite.ds.UpsertExternalNetworkEntity(ctx, entity1)
	suite.NoError(err)

	// Valid entity-no name
	entity2 := &storage.NetworkEntity{
		Info: &storage.NetworkEntityInfo{
			Id:   entity2ID.String(),
			Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
			Desc: &storage.NetworkEntityInfo_ExternalSource_{
				ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
					Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
						Cidr: "192.0.2.0/30",
					},
				},
			},
		},
		Scope: &storage.NetworkEntity_Scope{
			ClusterId: cluster1,
		},
	}
	err = suite.ds.UpsertExternalNetworkEntity(ctx, entity2)
	suite.NoError(err)

	// Invalid external source-invalid network
	entity3 := &storage.NetworkEntity{
		Info: &storage.NetworkEntityInfo{
			Id:   entity3ID.String(),
			Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
			Desc: &storage.NetworkEntityInfo_ExternalSource_{
				ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
					Name: "cidr1",
					Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
						Cidr: "300.0.2.0/24",
					},
				},
			},
		},
		Scope: &storage.NetworkEntity_Scope{
			ClusterId: cluster1,
		},
	}
	err = suite.ds.UpsertExternalNetworkEntity(ctx, entity3)
	suite.Error(err)

	// Invalid external source-invalid type
	entity4 := &storage.NetworkEntity{
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
	}
	err = suite.ds.UpsertExternalNetworkEntity(ctx, entity4)
	suite.Error(err)

	// Valid entity
	entity5 := &storage.NetworkEntity{
		Info: &storage.NetworkEntityInfo{
			Id:   entity5ID.String(),
			Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
			Desc: &storage.NetworkEntityInfo_ExternalSource_{
				ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
					Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
						Cidr: "192.0.2.0/24",
					},
				},
			},
		},
		Scope: &storage.NetworkEntity_Scope{
			ClusterId: cluster2,
		},
	}
	err = suite.ds.UpsertExternalNetworkEntity(ctx, entity5)
	suite.NoError(err)

	// Valid entity-update CIDR block
	entity5 = &storage.NetworkEntity{
		Info: &storage.NetworkEntityInfo{
			Id:   entity5.GetInfo().GetId(),
			Type: entity5.GetInfo().GetType(),
			Desc: &storage.NetworkEntityInfo_ExternalSource_{
				ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
					Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
						Cidr: "192.0.2.0/29",
					},
				},
			},
		},
		Scope: entity5.GetScope(),
	}
	err = suite.ds.UpsertExternalNetworkEntity(ctx, entity5)
	suite.NoError(err)

	// Invalid entity-CIDR already exists in cluster
	entity6 := &storage.NetworkEntity{
		Info: &storage.NetworkEntityInfo{
			Id:   entity6ID.String(),
			Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
			Desc: &storage.NetworkEntityInfo_ExternalSource_{
				ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
					Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
						Cidr: "192.0.2.0/29",
					},
				},
			},
		},
		Scope: &storage.NetworkEntity_Scope{
			ClusterId: cluster2,
		},
	}
	err = suite.ds.UpsertExternalNetworkEntity(ctx, entity6)
	suite.Error(err)

	// Invalid entity-invalid scope
	entity7 := &storage.NetworkEntity{
		Info: &storage.NetworkEntityInfo{
			Id:   entity7ID.String(),
			Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
			Desc: &storage.NetworkEntityInfo_ExternalSource_{
				ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
					Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
						Cidr: "192.0.2.0/24",
					},
				},
			},
		},
		Scope: &storage.NetworkEntity_Scope{
			ClusterId: "",
		},
	}
	err = suite.ds.UpsertExternalNetworkEntity(ctx, entity7)
	suite.Error(err)

	// Test Get
	actual, found, err := suite.ds.GetEntity(ctx, entity1.GetInfo().GetId())
	suite.NoError(err)
	suite.True(found)
	suite.Equal(entity1, actual)

	actual, found, err = suite.ds.GetEntity(ctx, entity5.GetInfo().GetId())
	suite.NoError(err)
	suite.True(found)
	suite.Equal(entity5.GetInfo().GetExternalSource().GetCidr(), actual.GetInfo().GetExternalSource().GetName())

	actual, found, err = suite.ds.GetEntity(ctx, entity3.GetInfo().GetId())
	suite.NoError(err)
	suite.False(found)
	suite.Nil(actual)

	actual, found, err = suite.ds.GetEntity(ctx, entity4.GetInfo().GetId())
	suite.NoError(err)
	suite.False(found)
	suite.Nil(actual)

	_, found, err = suite.ds.GetEntity(ctx, entity5.GetInfo().GetId())
	suite.NoError(err)
	suite.True(found)

	// Test Remove
	err = suite.ds.DeleteExternalNetworkEntity(ctx, entity1.GetInfo().GetId())
	suite.NoError(err)
	err = suite.ds.DeleteExternalNetworkEntity(ctx, entity2.GetInfo().GetId())
	suite.NoError(err)
	err = suite.ds.DeleteExternalNetworkEntity(ctx, entity5.GetInfo().GetId())
	suite.NoError(err)

	// Test GetAll
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: false}, nil)
	entities, err := suite.ds.GetAllEntities(ctx)
	suite.NoError(err)
	suite.Len(entities, 0)
}

func (suite *NetworkEntityDataStoreTestSuite) TestSAC() {
	cluster1 := "cluster1"
	cluster2 := "cluster2"

	entity1ID, _ := sac.NewClusterScopeResourceID(cluster1, uuid.NewV4().String())
	entity2ID, _ := sac.NewClusterScopeResourceID(cluster1, uuid.NewV4().String())
	entity3ID, _ := sac.NewClusterScopeResourceID(cluster2, uuid.NewV4().String())
	entity4ID, _ := sac.NewClusterScopeResourceID(cluster2, uuid.NewV4().String())

	entity1 := &storage.NetworkEntity{
		Info: &storage.NetworkEntityInfo{
			Id:   entity1ID.String(),
			Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
			Desc: &storage.NetworkEntityInfo_ExternalSource_{
				ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
					Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
						Cidr: "192.0.2.0/24",
					},
				},
			},
		},
		Scope: &storage.NetworkEntity_Scope{
			ClusterId: cluster1,
		},
	}

	entity2 := &storage.NetworkEntity{
		Info: &storage.NetworkEntityInfo{
			Id:   entity2ID.String(),
			Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
			Desc: &storage.NetworkEntityInfo_ExternalSource_{
				ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
					Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
						Cidr: "192.0.2.0/29",
					},
				},
			},
		},
		Scope: &storage.NetworkEntity_Scope{
			ClusterId: cluster1,
		},
	}

	entity3 := entity1.Clone()
	entity3.Info.Id = entity3ID.String()
	entity3.Scope.ClusterId = cluster2

	entity4 := entity2.Clone()
	entity4.Info.Id = entity4ID.String()
	entity4.Scope.ClusterId = cluster2

	noAccessCtx := sac.WithNoAccess(context.Background())
	cluster1ReadCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph),
			sac.ClusterScopeKeys(cluster1)))
	allClusterReadCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph),
			sac.ClusterScopeKeys(cluster1, cluster2)))
	cluster2WriteCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph),
			sac.ClusterScopeKeys(cluster2)))
	allClusterWriteCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph),
			sac.ClusterScopeKeys(cluster1, cluster2)))

	// Error-no access
	err := suite.ds.UpsertExternalNetworkEntity(noAccessCtx, entity1)
	suite.Error(err)

	// Error-cluster2 permissions tried to write cluster1 resource
	err = suite.ds.UpsertExternalNetworkEntity(cluster2WriteCtx, entity1)
	suite.Error(err)

	// No error-all cluster access
	err = suite.ds.UpsertExternalNetworkEntity(allClusterWriteCtx, entity2)
	suite.NoError(err)

	// No error-cluster2 access
	err = suite.ds.UpsertExternalNetworkEntity(cluster2WriteCtx, entity3)
	suite.NoError(err)

	// No error-all cluster access
	err = suite.ds.UpsertExternalNetworkEntity(allClusterWriteCtx, entity4)
	suite.NoError(err)

	// No access
	_, found, err := suite.ds.GetEntity(noAccessCtx, entity1.GetInfo().GetId())
	suite.NoError(err)
	suite.False(found)

	// Success-cluster1 permissions used to read cluster1 resource
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: false}, nil)
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
	suite.Len(actuals, 1)
	suite.Equal(entity2, actuals[0])

	// All resources accessible
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: false}, nil)
	actuals, err = suite.ds.GetAllEntities(allClusterReadCtx)
	suite.NoError(err)
	suite.Len(actuals, 3)

	// No access-cluster1 read permissions used to delete cluster1 resource
	err = suite.ds.DeleteExternalNetworkEntity(cluster1ReadCtx, entity2.GetInfo().GetId())
	suite.Error(err)

	// No access-cluster1 permissions used to delete cluster2 resource
	err = suite.ds.DeleteExternalNetworkEntity(cluster1ReadCtx, entity3.GetInfo().GetId())
	suite.Error(err)

	// Success
	err = suite.ds.DeleteExternalNetworkEntitiesForCluster(cluster2WriteCtx, cluster2)
	suite.NoError(err)

	// Success
	err = suite.ds.DeleteExternalNetworkEntitiesForCluster(allClusterWriteCtx, cluster1)
	suite.NoError(err)

	// Test GetAll
	entities, err := suite.ds.GetAllEntities(allClusterReadCtx)
	suite.NoError(err)
	suite.Len(entities, 0)
}

func (suite *NetworkEntityDataStoreTestSuite) TestDefaultGraphSetting() {
	ctx := sac.WithAllAccess(context.Background())
	cluster1 := "cluster1"

	entity1ID, _ := sac.NewClusterScopeResourceID(cluster1, uuid.NewV4().String())
	entity2ID, _ := sac.NewClusterScopeResourceID(cluster1, uuid.NewV4().String())

	entity1 := &storage.NetworkEntity{
		Info: &storage.NetworkEntityInfo{
			Id:   entity1ID.String(),
			Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
			Desc: &storage.NetworkEntityInfo_ExternalSource_{
				ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
					Name: "cidr1",
					Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
						Cidr: "192.0.2.0/24",
					},
					Default: true,
				},
			},
		},
		Scope: &storage.NetworkEntity_Scope{
			ClusterId: cluster1,
		},
	}
	err := suite.ds.UpsertExternalNetworkEntity(ctx, entity1)
	suite.NoError(err)

	entity2 := &storage.NetworkEntity{
		Info: &storage.NetworkEntityInfo{
			Id:   entity2ID.String(),
			Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
			Desc: &storage.NetworkEntityInfo_ExternalSource_{
				ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
					Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
						Cidr: "192.0.2.0/30",
					},
				},
			},
		},
		Scope: &storage.NetworkEntity_Scope{
			ClusterId: cluster1,
		},
	}
	err = suite.ds.UpsertExternalNetworkEntity(ctx, entity2)
	suite.NoError(err)

	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: true}, nil)
	actual, err := suite.ds.GetAllEntities(ctx)
	suite.NoError(err)
	suite.Len(actual, 1)

	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: false}, nil)
	actual, err = suite.ds.GetAllEntities(ctx)
	suite.NoError(err)
	suite.Len(actual, 2)

	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: true}, nil)
	actual, err = suite.ds.GetAllEntitiesForCluster(ctx, cluster1)
	suite.NoError(err)
	suite.Len(actual, 1)

	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: false}, nil)
	actual, err = suite.ds.GetAllEntitiesForCluster(ctx, cluster1)
	suite.NoError(err)
	suite.Len(actual, 2)

	err = suite.ds.DeleteExternalNetworkEntitiesForCluster(sac.WithAllAccess(context.Background()), cluster1)
	suite.NoError(err)
	err = suite.ds.DeleteExternalNetworkEntitiesForCluster(sac.WithAllAccess(context.Background()), cluster1)
	suite.NoError(err)
}
