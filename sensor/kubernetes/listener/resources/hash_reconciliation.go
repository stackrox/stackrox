package resources

import (
	"github.com/stackrox/rox/pkg/reconcile"
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

// ProcessHashes orchestrates the sensor-side reconciliation after a reconnect. It returns a map of events that
// should be sent back to Central to ensure that the states in Sensor and Central are in sync.
func (hr *InMemoryStoreReconciler) ProcessHashes(h map[string]uint64) map[string]reconcile.SensorReconciliationEvent {
	events := make(map[string]reconcile.SensorReconciliationEvent)
	for typeWithID, hashValue := range h {
		resType, resID := stringutils.Split2(typeWithID, ":")
		if resID == "" {
			log.Errorf("malformed hash key: %s", typeWithID)
			continue
		}
		resEvents, err := hr.storeProvider.Reconcile(resType, resID, hashValue)
		if err != nil {
			log.Errorf("reconciliation error: %s", err)
		}
		if resEvents == nil {
			log.Error("empty reconciliation result")
			continue
		}
		for ek, ev := range resEvents {
			events[ek] = ev
		}
	}
	return events
}
