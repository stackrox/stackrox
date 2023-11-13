package deduper

import (
	"sync/atomic"

	"github.com/stackrox/rox/pkg/deduperkey"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

// ObservationSet holds IDs that were seen during the initial sensor sync.
type ObservationSet struct {
	innerSet  set.Set[string]
	innerLock sync.Mutex
	open      *atomic.Bool
}

// NewObservationSet creates an empty sync observation set.
func NewObservationSet() *ObservationSet {
	open := atomic.Bool{}
	open.Store(true)
	return &ObservationSet{
		innerSet:  set.NewSet[string](),
		innerLock: sync.Mutex{},
		open:      &open,
	}
}

// LogObserved parses a key `k` and adds to the observation set.
func (s *ObservationSet) LogObserved(k deduperkey.Key) {
	if s.open.Load() {
		stringKey := k.String()
		s.innerLock.Lock()
		defer s.innerLock.Unlock()
		s.innerSet.Add(stringKey)
	}
}

// Close will stop processing of any further keys and return a list of parsed keys.
func (s *ObservationSet) Close() []string {
	s.open.Store(false)
	s.innerLock.Lock()
	defer s.innerLock.Unlock()

	return s.innerSet.AsSlice()
}
