package reconciliation

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/reconcile"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/deduper"
)

var (
	log = logging.LoggerForModule()
)

// Store the reconciliation store stores a map if resource types and ids.
// This allows Sensor to reconcile resources after connecting with Central.
type Store interface {
	reconcile.Reconcilable
	Cleanup()
	UpsertType(resType string)
	Upsert(resType, id string)
	Remove(resType, id string)
}

type store struct {
	lock          sync.Mutex
	resources     map[string]set.StringSet
	resourceTypes set.StringSet
}

// NewStore creates a new reconciliation Store
func NewStore() Store {
	return &store{
		resources:     make(map[string]set.StringSet),
		resourceTypes: set.NewStringSet(),
	}
}

func (s *store) Cleanup() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.resources = make(map[string]set.StringSet)
	for resType := range s.resourceTypes {
		s.resources[resType] = set.NewStringSet()
	}
}

func (s *store) ReconcileDelete(resType, resID string, _ uint64) (string, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if resType == deduper.TypeImageIntegration.String() {
		log.Infof("Reconcile Image Integration %s", resID)
	}
	if ids, found := s.resources[resType]; found {
		if ids.Contains(resID) {
			if resType == deduper.TypeImageIntegration.String() {
				log.Infof("Reconcile Image Integration FOUND %s", resID)
			}
			return "", nil
		}
		if resType == deduper.TypeImageIntegration.String() {
			log.Infof("Reconcile Image Integration NOT FOUND %s", resID)
		}
		return resID, nil
	}
	return "", errors.Errorf("resource type %s not supported", resType)
}

var _ Store = (*store)(nil)

// UpsertType a resource type. These types won't be removed if Cleanup is called.
func (s *store) UpsertType(resType string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.resourceTypes.Add(resType)
	s.resources[resType] = set.NewStringSet()
}

// Upsert a resource type and id
func (s *store) Upsert(resType, id string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if resType == deduper.TypeImageIntegration.String() {
		log.Infof("Upsert Image Integration: %s", id)
	}
	s.addResourceNoLock(resType, id)
}

// Remove a resource type and id
func (s *store) Remove(resType, id string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if resType == deduper.TypeImageIntegration.String() {
		log.Infof("Remove Image Integration: %s", id)
	}
	s.removeResourceNoLock(resType, id)
}

func (s *store) addResourceNoLock(resType, id string) {
	if ids, found := s.resources[resType]; !found {
		s.resources[resType] = set.NewStringSet(id)
		s.resourceTypes.Add(resType)
	} else {
		ids.Add(id)
	}
}

func (s *store) removeResourceNoLock(resType, id string) {
	if ids, found := s.resources[resType]; found {
		ids.Remove(id)
	}
}
