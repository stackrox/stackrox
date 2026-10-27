package store

import (
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()
)

// Store tracks MachineConfigPool resources and the node-role labels they select.
type Store struct {
	lock  sync.RWMutex
	pools map[string]map[string]string // MCP name -> matchLabels
}

// New returns a new Store.
func New() *Store {
	return &Store{
		pools: make(map[string]map[string]string),
	}
}

// AddOrUpdate upserts a MachineConfigPool entry.
func (s *Store) AddOrUpdate(name string, matchLabels map[string]string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	copied := make(map[string]string, len(matchLabels))
	for k, v := range matchLabels {
		copied[k] = v
	}
	s.pools[name] = copied
}

// Remove deletes a MachineConfigPool entry by name.
func (s *Store) Remove(name string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.pools, name)
}

// GetMatchLabels returns the matchLabels for a given MCP name, or nil if not found.
func (s *Store) GetMatchLabels(name string) map[string]string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	ml, ok := s.pools[name]
	if !ok {
		return nil
	}
	copied := make(map[string]string, len(ml))
	for k, v := range ml {
		copied[k] = v
	}
	return copied
}

// GetAll returns a copy of all stored MCP entries keyed by name.
func (s *Store) GetAll() map[string]map[string]string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	result := make(map[string]map[string]string, len(s.pools))
	for name, ml := range s.pools {
		copied := make(map[string]string, len(ml))
		for k, v := range ml {
			copied[k] = v
		}
		result[name] = copied
	}
	return result
}

// Cleanup deletes all entries from the store.
func (s *Store) Cleanup() {
	s.lock.Lock()
	defer s.lock.Unlock()
	clear(s.pools)
}
