package resources

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/stringutils"
)

// ResourceStoreReconciler handles sensor-side reconciliation using in-memory store
type ResourceStoreReconciler struct {
	storeProvider *InMemoryStoreProvider
}

// NewResourceStoreReconciler builds ResourceStoreReconciler for sensor-side reconciliation
func NewResourceStoreReconciler(storeProvider *InMemoryStoreProvider) *ResourceStoreReconciler {
	return &ResourceStoreReconciler{storeProvider: storeProvider}
}

// ProcessHashes orchestrates the sensor-side reconciliation after a reconnect. It returns a slice of resource IDs that
// should be deleted in Central to keep the state of Sensor and Central in sync.
func (hr *ResourceStoreReconciler) ProcessHashes(h map[string]uint64) []central.MsgFromSensor {
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
			continue
		}
		if toDeleteID == "" {
			log.Debug("empty reconciliation result - not found on Sensor")
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
	switch resType {
	case "Deployment":
		msg := central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     resID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_Deployment{
					Deployment: &storage.Deployment{Id: resID},
				},
			},
		}
		return &central.MsgFromSensor{Msg: &msg}, nil
	case "Pod":
		msg := central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     resID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_Pod{
					Pod: &storage.Pod{Id: resID},
				},
			},
		}
		return &central.MsgFromSensor{Msg: &msg}, nil
	default:
		return nil, errors.Errorf("Not implemented for resource type %v", resType)
	}
}
