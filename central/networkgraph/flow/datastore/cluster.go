package datastore

import (
	"context"
	"testing"

	graphConfigDS "github.com/stackrox/rox/central/networkgraph/config/datastore"
	"github.com/stackrox/rox/central/networkgraph/entity/networktree"
	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store"
	pgStore "github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/postgres"
)

// ClusterDataStore stores the network edges per cluster.
//
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

// GetTestPostgresClusterDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresClusterDataStore(t testing.TB, pool postgres.DB) (ClusterDataStore, error) {
	dbstore := pgStore.NewClusterStore(pool)
	configStore, err := graphConfigDS.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	networkTreeMgr := networktree.Singleton()
	entitiesByCluster := map[string][]*storage.NetworkEntityInfo{}
	err = networkTreeMgr.Initialize(entitiesByCluster)
	if err != nil {
		return nil, err
	}
	return NewClusterDataStore(dbstore, configStore, networkTreeMgr, nil), nil
}
