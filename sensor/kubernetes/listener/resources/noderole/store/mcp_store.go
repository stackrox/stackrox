package store

import (
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

// MachineConfigPoolStore tracks the set of MachineConfigPool names present in the cluster.
type MachineConfigPoolStore struct {
	lock     sync.RWMutex
	mcpNames set.StringSet
}

// NewMachineConfigPoolStore returns a new MachineConfigPoolStore.
func NewMachineConfigPoolStore() *MachineConfigPoolStore {
	return &MachineConfigPoolStore{
		mcpNames: set.NewStringSet(),
	}
}

// Add inserts an MCP name. Returns true if it was not already present.
func (s *MachineConfigPoolStore) Add(name string) bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.mcpNames.Add(name)
}

// Remove deletes an MCP name. Returns true if it was present.
func (s *MachineConfigPoolStore) Remove(name string) bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.mcpNames.Remove(name)
}

// GetRoles returns a snapshot of all MCP names currently tracked.
func (s *MachineConfigPoolStore) GetRoles() []string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.mcpNames.AsSlice()
}

// Cleanup deletes all entries from the store.
func (s *MachineConfigPoolStore) Cleanup() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.mcpNames = set.NewStringSet()
}
