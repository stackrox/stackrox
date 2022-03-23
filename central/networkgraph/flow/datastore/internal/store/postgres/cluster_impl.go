package postgres

import (
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store"
	"golang.org/x/net/context"
)

// NewClusterStore returns a new ClusterStore instance using the provided rocksdb instance.
func NewClusterStore(db *pgxpool.Pool) store.ClusterStore {
	log.Info("SHREWS => NewClusterStore")
	return &clusterStoreImpl{
		db: db,
	}
}

type clusterStoreImpl struct {
	db *pgxpool.Pool
}

// GetFlowStore returns the FlowStore for the cluster ID, or nil if none exists.
func (s *clusterStoreImpl) GetFlowStore(clusterID string) store.FlowStore {
	log.Infof("SHREWS => GetFlowStore => %s", clusterID)
	return &flowStoreImpl{
		db:        s.db,
		clusterID: clusterID,
	}
}

// CreateFlowStore returns the FlowStore for the cluster ID, or creates one if none exists.
func (s *clusterStoreImpl) CreateFlowStore(ctx context.Context, clusterID string) (store.FlowStore, error) {
	log.Infof("SHREWS => CreateFlowStore => %s", clusterID)

	fs := New(ctx, s.db, clusterID)
	//fs := &flowStoreImpl{
	//	db:        s.db,
	//	clusterID: clusterID,
	//}
	log.Infof("SHREWS => CreateFlowStore => %s", fs)
	return fs, nil
}
