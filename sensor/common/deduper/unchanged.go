package deduper

import (
	"sync/atomic"

	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

// ClosableSet holds IDs that were seen during the initial sensor sync.
type ClosableSet struct {
	innerSet  set.Set[string]
	innerLock sync.Mutex
	open      *atomic.Bool
}

// NewClosableSet creates an empty sync observation set.
func NewClosableSet() *ClosableSet {
	open := atomic.Bool{}
	open.Store(true)
	return &ClosableSet{
		innerSet:  set.NewSet[string](),
		innerLock: sync.Mutex{},
		open:      &open,
	}
}

// AddIfOpen parses a key `k` and adds to the observation set.
func (s *ClosableSet) AddIfOpen(stringKey string) {
	if s.open.Load() {
		s.innerLock.Lock()
		defer s.innerLock.Unlock()
		s.innerSet.Add(stringKey)
	}
}

// Close will stop processing of any further keys and return a list of parsed keys.
func (s *ClosableSet) Close() []string {
	s.open.Store(false)
	s.innerLock.Lock()
	defer s.innerLock.Unlock()

	return s.innerSet.AsSlice()
}
