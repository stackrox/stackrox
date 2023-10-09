package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/stringutils"
)

// InMemoryStoreReconciler handles sensor-side reconciliation using in-memory store
type ResourceStoreReconciler struct {
	storeProvider *InMemoryStoreProvider
}

// NewInMemoryStoreReconciler builds InMemoryStoreReconciler for sensor-side reconciliation
func NewInMemoryStoreReconciler(storeProvider *InMemoryStoreProvider) *InMemoryStoreReconciler {
	return &InMemoryStoreReconciler{storeProvider: storeProvider}
}

// ProcessHashes orchestrates the sensor-side reconciliation after a reconnect. It returns a slice of resource IDs that
// should be deleted in Central to keep the state of Sensor and Central in sync.
func (hr *InMemoryStoreReconciler) ProcessHashes(h map[string]uint64) []central.MsgFromSensor {
	events := make([]central.MsgFromSensor, 0)
	for typeWithID, hashValue := range h {
		resType, resID := stringutils.Split2(typeWithID, ":")
		if resID == "" {
			log.Errorf("malformed hash key: %s", typeWithID)
			continue
		}
		toDeleteID, err := hr.storeProvider.ReconcileDelete(resType, resID, hashValue)
		if err != nil {
			log.Errorf("reconciliation error: %s", err)
		}
		if toDeleteID == "" {
			log.Error("empty reconciliation result")
			continue
		}
		delMsg, err := resourceToMessage(resType, toDeleteID)
		if err != nil {
			log.Errorf("converting resource to MsgFromSensor: %s", err)
			continue
		}
		events = append(events, *delMsg)
	}
	return events
}

func resourceToMessage(resType string, resID string) (*central.MsgFromSensor, error) {
	_, _ = resType, resID
	panic("Not implemented")

}
