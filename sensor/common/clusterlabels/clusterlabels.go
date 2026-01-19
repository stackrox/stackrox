package clusterlabels

import (
	"github.com/stackrox/rox/pkg/sync"
)

type Store struct {
	mutex         sync.RWMutex
	clusterLabels map[string]string
}

// NewStore creates a new cluster labels store
func NewStore() *Store {
	return &Store{clusterLabels: make(map[string]string)}
}

func (s *Store) Get() map[string]string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.clusterLabels
}

func (s *Store) Set(labels map[string]string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.clusterLabels = labels
}
