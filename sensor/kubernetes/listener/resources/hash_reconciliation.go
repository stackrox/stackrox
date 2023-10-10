package resources

import (
	"reflect"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

// FIXME(ROX-19696): This is a temporary test interface. Remove in favor of PR #8006
type Key struct {
	ID   string
	Type reflect.Type
}

var (
	TypeNetworkPolicy                        = reflect.TypeOf(&central.SensorEvent_NetworkPolicy{})
	TypeDeployment                           = reflect.TypeOf(&central.SensorEvent_Deployment{})
	TypePod                                  = reflect.TypeOf(&central.SensorEvent_Pod{})
	TypeNamespace                            = reflect.TypeOf(&central.SensorEvent_Namespace{})
	TypeSecret                               = reflect.TypeOf(&central.SensorEvent_Secret{})
	TypeNode                                 = reflect.TypeOf(&central.SensorEvent_Node{})
	TypeNodeInventory                        = reflect.TypeOf(&central.SensorEvent_NodeInventory{})
	TypeServiceAccount                       = reflect.TypeOf(&central.SensorEvent_ServiceAccount{})
	TypeRole                                 = reflect.TypeOf(&central.SensorEvent_Role{})
	TypeBinding                              = reflect.TypeOf(&central.SensorEvent_Binding{})
	TypeProcessIndicator                     = reflect.TypeOf(&central.SensorEvent_ProcessIndicator{})
	TypeProviderMetadata                     = reflect.TypeOf(&central.SensorEvent_ProviderMetadata{})
	TypeOrchestratorMetadata                 = reflect.TypeOf(&central.SensorEvent_OrchestratorMetadata{})
	TypeImageIntegration                     = reflect.TypeOf(&central.SensorEvent_ImageIntegration{})
	TypeComplianceOperatorResult             = reflect.TypeOf(&central.SensorEvent_ComplianceOperatorResult{})
	TypeComplianceOperatorProfile            = reflect.TypeOf(&central.SensorEvent_ComplianceOperatorProfile{})
	TypeComplianceOperatorRule               = reflect.TypeOf(&central.SensorEvent_ComplianceOperatorRule{})
	TypeComplianceOperatorScanSettingBinding = reflect.TypeOf(&central.SensorEvent_ComplianceOperatorScanSettingBinding{})
	TypeComplianceOperatorScan               = reflect.TypeOf(&central.SensorEvent_ComplianceOperatorScan{})
)

// FIXME(ROX-19696)

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
func (hr *ResourceStoreReconciler) ProcessHashes(h map[Key]uint64) []central.MsgFromSensor {
	events := make([]central.MsgFromSensor, 0)
	for hash, hashValue := range h {
		toDeleteID, err := hr.storeProvider.ReconcileDelete(hash.Type.String(), hash.ID, hashValue)
		if err != nil {
			log.Errorf("reconciliation error: %s", err)
			continue
		}
		if toDeleteID == "" {
			log.Debug("empty reconciliation result - not found on Sensor")
			continue
		}
		delMsg, err := resourceToMessage(hash.Type.String(), toDeleteID)
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
	case TypeDeployment.String():
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
	case TypePod.String():
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
