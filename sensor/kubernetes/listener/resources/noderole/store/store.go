package store

import (
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

// Store tracks the set of MachineConfigPool names present in the cluster.
type Store struct {
	lock  sync.RWMutex
	roles set.StringSet
}

// New returns a new Store.
func New() *Store {
	return &Store{
		roles: set.NewStringSet(),
	}
}

// Add inserts an MCP name. Returns true if it was not already present.
func (s *Store) Add(name string) bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.roles.Add(name)
}

// Remove deletes an MCP name. Returns true if it was present.
func (s *Store) Remove(name string) bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.roles.Remove(name)
}

// GetRoles returns a snapshot of all MCP names currently tracked.
func (s *Store) GetRoles() []string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.roles.AsSlice()
}

// Cleanup deletes all entries from the store.
func (s *Store) Cleanup() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.roles = set.NewStringSet()
}
