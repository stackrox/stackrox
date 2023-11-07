package deduper

import (
	"sync/atomic"

	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

// ClosableSet is a concurrency-safe set of strings. Items can be added to the set as long as the set is open.
// Closing the set ignores all future `Add` operations but allows to fetch already stored items.
type ClosableSet struct {
	innerSet  set.Set[string]
	innerLock sync.Mutex
	open      *atomic.Bool
}

// NewClosableSet creates an empty ClosableSet.
func NewClosableSet() *ClosableSet {
	open := atomic.Bool{}
	open.Store(true)
	return &ClosableSet{
		innerSet:  set.NewSet[string](),
		innerLock: sync.Mutex{},
		open:      &open,
	}
}

// AddIfOpen adds item to an unclosed set. If set is closed, this operation does nothing.
func (s *ClosableSet) AddIfOpen(stringKey string) {
	if s.open.Load() {
		s.innerLock.Lock()
		defer s.innerLock.Unlock()
		s.innerSet.Add(stringKey)
	}
}

// Close marks set as closed preventing any new items to be added. It returns the currently stored items.
func (s *ClosableSet) Close() []string {
	s.open.Store(false)
	s.innerLock.Lock()
	defer s.innerLock.Unlock()

	return s.innerSet.AsSlice()
}
