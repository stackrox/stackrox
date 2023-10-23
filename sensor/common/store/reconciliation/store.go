package reconciliation

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/reconcile"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

type Store interface {
	reconcile.Reconcilable
	Cleanup()
	Add(resType, id string)
	Remove(resType, id string)
}

type store struct {
	lock      sync.Mutex
	resources map[string]set.StringSet
}

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

func (s *store) Add(resType, id string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.addResourceNoLock(resType, id)
}

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
