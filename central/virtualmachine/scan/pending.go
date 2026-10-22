package scan

import "github.com/stackrox/rox/pkg/sync"

var pending = &Tracker{}

// Tracker records scans currently being processed by Central. Entries exist only
// for the lifetime of an index report, so failures and restarts leave no stale state.
type Tracker struct {
	pending sync.Map
}

// Singleton returns the tracker shared by VM ingestion and the VM API.
func Singleton() *Tracker {
	return pending
}

// Start marks a VM scan as pending and returns a function that clears it.
// Pending status is best effort when reports overlap during a reconnect.
func (t *Tracker) Start(vmID string) func() {
	t.pending.Store(vmID, struct{}{})
	return func() {
		t.pending.Delete(vmID)
	}
}

// IsPending reports whether Central is processing a scan for this VM.
func (t *Tracker) IsPending(vmID string) bool {
	if t == nil {
		return false
	}
	_, ok := t.pending.Load(vmID)
	return ok
}
