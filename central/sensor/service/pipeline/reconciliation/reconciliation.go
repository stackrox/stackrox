package reconciliation

import (
	"github.com/stackrox/rox/pkg/set"
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
	synced bool
	idSet  *set.StringSet
}

// Add adds an id if the store has not already been closed
func (s *storeImpl) Add(id string) {
	if s.idSet != nil {
		s.idSet.Add(id)
	}
}

// GetSet returns the string set of IDs
func (s *storeImpl) GetSet() set.StringSet {
	return *s.idSet
}

// Close deallocates the internal set
func (s *storeImpl) Close() {
	s.idSet = nil
}
