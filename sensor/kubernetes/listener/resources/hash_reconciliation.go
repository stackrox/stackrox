package resources

import (
	"github.com/stackrox/rox/pkg/stringutils"
)

// InMemoryStoreReconciler handles sensor-side reconciliation using in-memory store
type InMemoryStoreReconciler struct {
	storeProvider *InMemoryStoreProvider
}

// NewInMemoryStoreReconciler builds InMemoryStoreReconciler for sensor-side reconciliation
func NewInMemoryStoreReconciler(storeProvider *InMemoryStoreProvider) *InMemoryStoreReconciler {
	return &InMemoryStoreReconciler{storeProvider: storeProvider}
}

// ProcessHashes orchestrates the sensor-side reconciliation after a reconnect. It returns a slice of resource IDs that
// should be deleted in Central to keep the state of Sensor and Central in sync.
func (hr *InMemoryStoreReconciler) ProcessHashes(h map[string]uint64) []string {
	events := make([]string, 0)
	for typeWithID, hashValue := range h {
		resType, resID := stringutils.Split2(typeWithID, ":")
		if resID == "" {
			log.Errorf("malformed hash key: %s", typeWithID)
			continue
		}
		resEvents, err := hr.storeProvider.ReconcileDelete(resType, resID, hashValue)
		if err != nil {
			log.Errorf("reconciliation error: %s", err)
		}
		if resEvents == nil {
			log.Error("empty reconciliation result")
			continue
		}
		events = append(events, resEvents...)
	}
	return events
}
