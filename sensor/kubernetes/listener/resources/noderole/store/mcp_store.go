package store

import (
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

// MachineConfigPoolStore tracks the node roles targeted by the
// MachineConfigPools present in the cluster, keyed by pool name. Roles are
// stored per pool so that a role contributed by several pools remains tracked
// as long as one of those pools exists.
type MachineConfigPoolStore struct {
	lock  sync.RWMutex
	pools map[string]set.StringSet
}

// NewMachineConfigPoolStore returns a new MachineConfigPoolStore.
func NewMachineConfigPoolStore() *MachineConfigPoolStore {
	return &MachineConfigPoolStore{
		pools: make(map[string]set.StringSet),
	}
}

// Upsert records the roles targeted by the named MachineConfigPool, replacing
// any roles previously recorded for that pool. It returns true if the overall
// set of roles changed.
func (s *MachineConfigPoolStore) Upsert(name string, roles set.StringSet) bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	before := s.rolesNoLock()
	s.pools[name] = roles
	return !before.Equal(s.rolesNoLock())
}

// Remove deletes the entry of the named MachineConfigPool. It returns true if
// the overall set of roles changed.
func (s *MachineConfigPoolStore) Remove(name string) bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	before := s.rolesNoLock()
	delete(s.pools, name)
	return !before.Equal(s.rolesNoLock())
}

// GetRoles returns a snapshot of all node roles currently tracked.
func (s *MachineConfigPoolStore) GetRoles() []string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.rolesNoLock().AsSlice()
}

// Cleanup deletes all entries from the store.
func (s *MachineConfigPoolStore) Cleanup() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.pools = make(map[string]set.StringSet)
}

// rolesNoLock returns the union of the roles of all tracked pools. The caller
// must hold the lock.
func (s *MachineConfigPoolStore) rolesNoLock() set.StringSet {
	roles := set.NewStringSet()
	for _, poolRoles := range s.pools {
		for role := range poolRoles {
			roles.Add(role)
		}
	}
	return roles
}
