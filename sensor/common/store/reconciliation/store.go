package reconciliation

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/reconcile"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

// Store the reconciliation store stores a map if resource types and ids.
// This allows Sensor to reconcile resources after connecting with Central.
type Store interface {
	reconcile.Reconcilable
	Cleanup()
	Upsert(resType, id string)
	Remove(resType, id string)
}

type store struct {
	lock      sync.Mutex
	resources map[string]set.StringSet
}

// NewStore creates a new reconciliation Store
func NewStore() Store {
	return &store{
		resources: make(map[string]set.StringSet),
	}
}

func (s *store) Cleanup() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.resources = make(map[string]set.StringSet)
}

func (s *store) ReconcileDelete(resType, resID string, _ uint64) (string, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if ids, found := s.resources[resType]; found {
		if ids.Contains(resID) {
			return "", nil
		}
		return resID, nil
	}
	return "", errors.Errorf("resource type %s not supported", resType)
}

var _ Store = (*store)(nil)

// Upsert a resource type and id
func (s *store) Upsert(resType, id string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.addResourceNoLock(resType, id)
}

// Remove a resource type and id
func (s *store) Remove(resType, id string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.removeResourceNoLock(resType, id)
}

func (s *store) addResourceNoLock(resType, id string) {
	if ids, found := s.resources[resType]; !found {
		s.resources[resType] = set.NewStringSet(id)
	} else {
		ids.Add(id)
	}
}

func (s *store) removeResourceNoLock(resType, id string) {
	if ids, found := s.resources[resType]; found {
		ids.Remove(id)
	}
	if len(s.resources[resType]) == 0 {
		delete(s.resources, resType)
	}
}
