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
	Close()
}

// NewStore returns a new Store
func NewStore() Store {
	s := set.NewStringSet()
	return &storeImpl{
		idSet: &s,
	}
}

type storeImpl struct {
	idSet *set.StringSet
	lock  sync.Mutex
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
func (s *storeImpl) Close() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.idSet = nil
}

// StoreMap is wrapper around a map of types -> reconciliation stores
type StoreMap struct {
	reconciliationMap map[string]Store
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

// Add adds an id to the type
func (s *StoreMap) Add(i interface{}, id string) {
	if s.reconciliationMap == nil {
		utils.Should(errors.Errorf("Attempted to perform an Add on a closed reconciliation store for the following ID: %s", id))
		return
	}
	typ := reflectutils.Type(i)
	val, ok := s.reconciliationMap[typ]
	if !ok {
		val = NewStore()
		s.reconciliationMap[typ] = val
	}
	val.Add(id)
}

// IsClosed indicates if the map is closed.
func (s *StoreMap) IsClosed() bool {
	return s.reconciliationMap == nil
}

// Close closes all of the references stores and the map itself.
func (s *StoreMap) Close() {
	if s.reconciliationMap == nil {
		return
	}
	for _, store := range s.reconciliationMap {
		store.Close()
	}
	s.reconciliationMap = nil
}
