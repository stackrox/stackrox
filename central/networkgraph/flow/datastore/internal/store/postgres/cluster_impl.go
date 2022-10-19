package postgres

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store"
	"github.com/stackrox/rox/pkg/sync"
)

// NewClusterStore returns a new ClusterStore instance using the provided rocksdb instance.
func NewClusterStore(db *pgxpool.Pool) store.ClusterStore {
	return &clusterStoreImpl{
		db:        db,
		flowStore: make(map[string]store.FlowStore),
	}
}

type clusterStoreImpl struct {
	db        *pgxpool.Pool
	lock      sync.Mutex
	flowStore map[string]store.FlowStore
}

// GetFlowStore returns the FlowStore for the cluster ID, or nil if none exists.
func (s *clusterStoreImpl) GetFlowStore(clusterID string) store.FlowStore {
	s.lock.Lock()
	defer s.lock.Unlock()

	flowStore, found := s.flowStore[clusterID]
	if !found || flowStore == nil {
		flowStore = New(s.db, clusterID)
		s.flowStore[clusterID] = flowStore
	}
	return flowStore
}

// CreateFlowStore returns the FlowStore for the cluster ID, or creates one if none exists.
func (s *clusterStoreImpl) CreateFlowStore(_ context.Context, clusterID string) (store.FlowStore, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	flowStore := New(s.db, clusterID)
	s.flowStore[clusterID] = flowStore
	return flowStore, nil
}
