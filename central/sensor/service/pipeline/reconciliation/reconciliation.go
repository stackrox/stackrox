package reconciliation

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/reflectutils"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

// Store is an interface for reconciliation
type Store interface {
	Add(id string)
	GetSet() set.StringSet
	Close(numObjectsDeleted int)
	GetNumObjectsDeleted() (int, bool)
}

// NewStore returns a new Store
func NewStore() Store {
	s := set.NewStringSet()
	return &storeImpl{
		idSet: &s,
	}
}

type storeImpl struct {
	numObjectsDeleted int
	idSet             *set.StringSet
	lock              sync.Mutex
}

// Add adds an id if the store has not already been closed
func (s *storeImpl) Add(id string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.idSet != nil {
		s.idSet.Add(id)
	}
}

// GetSet returns the string set of IDs
func (s *storeImpl) GetSet() set.StringSet {
	s.lock.Lock()
	defer s.lock.Unlock()
	return *s.idSet
}

// Close deallocates the internal set
func (s *storeImpl) Close(numObjectsDeleted int) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.numObjectsDeleted = numObjectsDeleted
	s.idSet = nil
}

// GetNumObjectsDeleted returns the number of objects deleted by the store,
// and a bool indicating whether the store is closed.
func (s *storeImpl) GetNumObjectsDeleted() (int, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.idSet != nil {
		return 0, false
	}
	return s.numObjectsDeleted, true
}

// StoreMap is wrapper around a map of types -> reconciliation stores
type StoreMap struct {
	reconciliationMap map[string]Store

	deletedElementsByTypeLock sync.RWMutex
	deletedElementsByType     map[string]int
}

// NewStoreMap creates a store map
func NewStoreMap() *StoreMap {
	return &StoreMap{
		reconciliationMap: make(map[string]Store),
	}
}

// Get retrieves the store for that type
// Never return nil to prevent accidental panics if nil checks are not performed.
func (s *StoreMap) Get(i interface{}) Store {
	if s.reconciliationMap == nil {
		utils.Should(errors.Errorf("Attempted to perform a Get on a closed reconciliation store for the following: %+v", i))
		return NewStore()
	}
	typ := reflectutils.Type(i)
	val, ok := s.reconciliationMap[typ]
	if !ok {
		val = NewStore()
		s.reconciliationMap[typ] = val
	}
	return val
}

// AddWithTypeString adds an id directly to type with name typeString
func (s *StoreMap) AddWithTypeString(typeString string, id string) {
	if s.reconciliationMap == nil {
		utils.Should(errors.Errorf("Attempted to perform an Add on a closed reconciliation store for the following ID: %s", id))
		return
	}
	val, ok := s.reconciliationMap[typeString]
	if !ok {
		val = NewStore()
		s.reconciliationMap[typeString] = val
	}
	val.Add(id)
}

// Add adds an id to the type
func (s *StoreMap) Add(i interface{}, id string) {
	typeString := reflectutils.Type(i)
	s.AddWithTypeString(typeString, id)
}

// IsClosed indicates if the map is closed.
func (s *StoreMap) IsClosed() bool {
	return s.reconciliationMap == nil
}

// DeletedElementsByType returns the number of elements deleted as part of reconciliation
// by type. It has a second return param which indicates whether reconciliation has been
// finished yet.
func (s *StoreMap) DeletedElementsByType() (map[string]int, bool) {
	s.deletedElementsByTypeLock.RLock()
	defer s.deletedElementsByTypeLock.RUnlock()
	if s.deletedElementsByType == nil {
		return nil, false
	}
	return s.deletedElementsByType, true
}

// Close closes all of the references stores and the map itself.
func (s *StoreMap) Close() {
	if s.reconciliationMap == nil {
		return
	}
	s.deletedElementsByTypeLock.Lock()
	defer s.deletedElementsByTypeLock.Unlock()
	s.deletedElementsByType = make(map[string]int, len(s.reconciliationMap))
	for typ, store := range s.reconciliationMap {
		numDeleted, storeClosed := store.GetNumObjectsDeleted()
		if storeClosed {
			s.deletedElementsByType[typ] = numDeleted
		} else {
			// We don't currently close all stores in the pipeline, so do it here.
			store.Close(0)
		}
	}
	log.Infof("Reconciliation done. Number of objects deleted by type: %v", s.deletedElementsByType)
	s.reconciliationMap = nil
}
