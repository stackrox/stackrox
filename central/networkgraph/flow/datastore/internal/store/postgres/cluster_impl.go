package postgres

import (
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store"
	"github.com/stackrox/rox/pkg/sync"
)

// NewClusterStore returns a new ClusterStore instance using the provided rocksdb instance.
func NewClusterStore(db *pgxpool.Pool) store.ClusterStore {
	return &clusterStoreImpl{
		db:             db,
		clusterMutexes: make(map[string]*sync.Mutex),
	}
}

type clusterStoreImpl struct {
	db               *pgxpool.Pool
	clusterMutexLock sync.Mutex
	clusterMutexes   map[string]*sync.Mutex
}

// GetFlowStore returns the FlowStore for the cluster ID, or nil if none exists.
func (s *clusterStoreImpl) GetFlowStore(clusterID string) store.FlowStore {
	s.clusterMutexLock.Lock()
	lock, ok := s.clusterMutexes[clusterID]
	if !ok {
		lock = &sync.Mutex{}
		s.clusterMutexes[clusterID] = lock
	}
	s.clusterMutexLock.Unlock()
	return &flowStoreImpl{
		db:        s.db,
		clusterID: clusterID,
		lock:      lock,
	}
}
