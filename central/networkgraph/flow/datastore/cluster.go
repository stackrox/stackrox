package datastore

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	graphConfigDS "github.com/stackrox/rox/central/networkgraph/config/datastore"
	"github.com/stackrox/rox/central/networkgraph/entity/networktree"
	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store"
	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/expiringcache"
	rocksdbBase "github.com/stackrox/rox/pkg/rocksdb"
	"gorm.io/gorm"
)

// ClusterDataStore stores the network edges per cluster.
//go:generate mockgen-wrapper
type ClusterDataStore interface {
	GetFlowStore(ctx context.Context, clusterID string) (FlowDataStore, error)
	CreateFlowStore(ctx context.Context, clusterID string) (FlowDataStore, error)
}

// NewClusterDataStore returns a new instance of ClusterDataStore using the input storage underneath.
func NewClusterDataStore(storage store.ClusterStore, graphConfig graphConfigDS.DataStore, networkTreeMgr networktree.Manager, deletedDeploymentsCache expiringcache.Cache) ClusterDataStore {
	return &clusterDataStoreImpl{
		storage:                 storage,
		graphConfig:             graphConfig,
		networkTreeMgr:          networkTreeMgr,
		deletedDeploymentsCache: deletedDeploymentsCache,
	}
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresClusterDataStore(ctx context.Context, t *testing.T, pool *pgxpool.Pool, gormDB *gorm.DB) ClusterDataStore {
	dbstore := postgres.NewClusterStore(pool)
	configStore := graphConfigDS.GetTestPostgresDataStore(ctx, t, pool, gormDB)
	networkTreeMgr := networktree.Singleton()
	entitiesByCluster := map[string][]*storage.NetworkEntityInfo{}
	networkTreeMgr.Initialize(entitiesByCluster)
	return NewClusterDataStore(dbstore, configStore, networkTreeMgr, nil)
}

// GetTestRocksBleveDataStore provides a datastore connected to rocksdb and bleve for testing purposes.
func GetTestRocksBleveClusterDataStore(t *testing.T, rocksengine *rocksdbBase.RocksDB) ClusterDataStore {
	dbstore := rocksdb.NewClusterStore(rocksengine)
	configStore := graphConfigDS.GetTestRocksBleveDataStore(t, rocksengine)
	networkTreeMgr := networktree.Singleton()
	entitiesByCluster := map[string][]*storage.NetworkEntityInfo{}
	networkTreeMgr.Initialize(entitiesByCluster)
	return NewClusterDataStore(dbstore, configStore, networkTreeMgr, nil)
}
