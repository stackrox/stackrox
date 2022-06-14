package datastore

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	graphConfigDS "github.com/stackrox/rox/central/networkgraph/config/datastore"
	"github.com/stackrox/rox/central/networkgraph/entity/networktree"
	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store"
	postgresStorage "github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store/postgres"
	rocksdbStorage "github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stretchr/testify/assert"
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

// GetTestPostgresDataStore provides a datastore hooked on postgres for testing purposes.
func GetTestPostgresDataStore(ctx context.Context, t *testing.T, pool *pgxpool.Pool, gormDB *gorm.DB) (ClusterDataStore, error) {
	dbstore := postgresStorage.NewClusterStore(pool)
	configStore, err := graphConfigDS.GetTestPostgresDataStore(ctx, t, pool, gormDB)
	assert.NoError(t, err)
	networkTreeMgr := networktree.Singleton()
	return NewClusterDataStore(dbstore, configStore, networkTreeMgr, nil), nil
}

// GetTestRocksBleveDataStore provides a processbaselineresult datastore hooked on rocksDB for testing purposes.
func GetTestRocksBleveDataStore(t *testing.T, rocksEngine *rocksdb.RocksDB) (ClusterDataStore, error) {
	dbstore := rocksdbStorage.NewClusterStore(rocksEngine)
	configStore, err := graphConfigDS.GetTestRocksBleveDataStore(t, rocksEngine)
	assert.NoError(t, err)
	networkTreeMgr := networktree.Singleton()
	return NewClusterDataStore(dbstore, configStore, networkTreeMgr, nil), nil
}
