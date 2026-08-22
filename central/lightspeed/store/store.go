package store

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/sync"
)

type entry struct {
	host string
	info *central.LightspeedInfo
}

// Store is an in-memory store for Lightspeed config and status per cluster.
type Store struct {
	mu   sync.RWMutex
	data map[string]*entry
}

// New returns a new Store instance.
func New() *Store {
	return &Store{
		data: make(map[string]*entry),
	}
}

// SetHost sets the Lightspeed host for the given cluster and resets info to nil.
func (s *Store) SetHost(clusterID, host string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if e := s.data[clusterID]; e != nil {
		e.host = host
		e.info = nil
	} else {
		s.data[clusterID] = &entry{host: host}
	}
}

// GetHost returns the Lightspeed host for the given cluster.
func (s *Store) GetHost(clusterID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if e := s.data[clusterID]; e != nil {
		return e.host
	}
	return ""
}

// UpdateInfo updates the Lightspeed info for the given cluster.
func (s *Store) UpdateInfo(clusterID string, info *central.LightspeedInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if e := s.data[clusterID]; e != nil {
		e.info = info
	} else {
		s.data[clusterID] = &entry{info: info}
	}
}

// Get returns both the host and info for the given cluster.
func (s *Store) Get(clusterID string) (host string, info *central.LightspeedInfo) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if e := s.data[clusterID]; e != nil {
		return e.host, e.info
	}
	return "", nil
}
