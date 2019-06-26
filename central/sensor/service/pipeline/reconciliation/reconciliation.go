package reconciliation

import (
	"github.com/stackrox/rox/pkg/reflectutils"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
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
func (s *StoreMap) Get(i interface{}) Store {
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
	typ := reflectutils.Type(i)
	val, ok := s.reconciliationMap[typ]
	if !ok {
		val = NewStore()
		s.reconciliationMap[typ] = val
	}
	val.Add(id)
}

// All returns all of the reconciliation stores
func (s *StoreMap) All() []Store {
	stores := make([]Store, 0, len(s.reconciliationMap))
	for _, s := range s.reconciliationMap {
		stores = append(stores, s)
	}
	return stores
}
