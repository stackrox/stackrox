package scan

import "github.com/stackrox/rox/pkg/sync"

var pending = &Tracker{counts: make(map[string]int)}

// Tracker records scans currently being processed by Central. Entries exist only
// for the lifetime of an index report, so failures and restarts leave no stale state.
type Tracker struct {
	mutex  sync.RWMutex
	counts map[string]int
}

// Singleton returns the tracker shared by VM ingestion and the VM API.
func Singleton() *Tracker {
	return pending
}

// Start marks a VM scan as pending and returns a function to call when processing
// finishes. Counting overlapping reports keeps the VM pending until all finish.
func (t *Tracker) Start(vmID string) func() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if t.counts == nil {
		t.counts = make(map[string]int)
	}
	t.counts[vmID]++
	var finished sync.Once
	return func() {
		finished.Do(func() {
			t.mutex.Lock()
			defer t.mutex.Unlock()
			t.counts[vmID]--
			if t.counts[vmID] == 0 {
				delete(t.counts, vmID)
			}
		})
	}
}

// IsPending reports whether Central is processing a scan for this VM.
func (t *Tracker) IsPending(vmID string) bool {
	if t == nil {
		return false
	}
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.counts[vmID] > 0
}
