package resources

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common/deduper"
)

// ResourceStoreReconciler handles sensor-side reconciliation using in-memory store
type ResourceStoreReconciler struct {
	storeProvider *StoreProvider
}

// NewResourceStoreReconciler builds ResourceStoreReconciler for sensor-side reconciliation
func NewResourceStoreReconciler(storeProvider *StoreProvider) *ResourceStoreReconciler {
	return &ResourceStoreReconciler{storeProvider: storeProvider}
}

// ProcessHashes orchestrates the sensor-side reconciliation after a reconnect. It returns a slice of Sensor messages that
// should be deleted in Central to keep the state of Sensor and Central in sync.
func (hr *ResourceStoreReconciler) ProcessHashes(h map[deduper.Key]uint64) []central.MsgFromSensor {
	events := make([]central.MsgFromSensor, 0)
	for hash, hashValue := range h {
		toDeleteID, err := hr.storeProvider.ReconcileDelete(hash.ResourceType.String(), hash.ID, hashValue)
		if err != nil {
			log.Errorf("reconciliation error: %s", err)
			continue
		}
		if toDeleteID == "" {
			log.Debug("empty reconciliation result - not found on Sensor")
			continue
		}
		delMsg, err := resourceToMessage(hash.ResourceType.String(), toDeleteID)
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
	case deduper.TypeDeployment.String():
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
	case deduper.TypePod.String():
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
	case deduper.TypeServiceAccount.String():
		msg := central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     resID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_ServiceAccount{
					ServiceAccount: &storage.ServiceAccount{Id: resID},
				},
			},
		}
		return &central.MsgFromSensor{Msg: &msg}, nil
	case deduper.TypeSecret.String():
		msg := central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     resID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_Secret{
					Secret: &storage.Secret{Id: resID},
				},
			},
		}
		return &central.MsgFromSensor{Msg: &msg}, nil
	case deduper.TypeNetworkPolicy.String():
		msg := central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     resID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_NetworkPolicy{
					NetworkPolicy: &storage.NetworkPolicy{Id: resID}},
			},
		}
		return &central.MsgFromSensor{Msg: &msg}, nil
	case deduper.TypeNode.String():
		msg := central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     resID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_Node{
					Node: &storage.Node{Id: resID},
				},
			},
		}
		return &central.MsgFromSensor{Msg: &msg}, nil
	case deduper.TypeRole.String():
		msg := central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     resID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_Role{
					Role: &storage.K8SRole{Id: resID},
				},
			},
		}
		return &central.MsgFromSensor{Msg: &msg}, nil
	case deduper.TypeBinding.String():
		msg := central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     resID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_Binding{
					Binding: &storage.K8SRoleBinding{Id: resID},
				},
			},
		}
		return &central.MsgFromSensor{Msg: &msg}, nil
	case deduper.TypeNamespace.String():
		msg := central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     resID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_Namespace{
					Namespace: &storage.NamespaceMetadata{Id: resID},
				},
			},
		}
		return &central.MsgFromSensor{Msg: &msg}, nil
	case deduper.TypeComplianceOperatorProfile.String():
		msg := central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     resID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorProfile{
					ComplianceOperatorProfile: &storage.ComplianceOperatorProfile{Id: resID},
				},
			},
		}
		return &central.MsgFromSensor{Msg: &msg}, nil
	case deduper.TypeComplianceOperatorResult.String():
		if features.ComplianceEnhancements.Enabled() {
			msg := central.MsgFromSensor_Event{
				Event: &central.SensorEvent{
					Id:     resID,
					Action: central.ResourceAction_REMOVE_RESOURCE,
					Resource: &central.SensorEvent_ComplianceOperatorResultV2{
						ComplianceOperatorResultV2: &central.ComplianceOperatorCheckResultV2{Id: resID},
					},
				},
			}
			return &central.MsgFromSensor{Msg: &msg}, nil
		}
		msg := central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     resID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorResult{
					ComplianceOperatorResult: &storage.ComplianceOperatorCheckResult{Id: resID},
				},
			},
		}
		return &central.MsgFromSensor{Msg: &msg}, nil
	case deduper.TypeComplianceOperatorRule.String():
		msg := central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     resID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorRule{
					ComplianceOperatorRule: &storage.ComplianceOperatorRule{
						Id: resID,
						Annotations: map[string]string{
							// This annotation is needed, otherwise central ignores this message
							v1alpha1.RuleIDAnnotationKey: v1alpha1.RuleIDAnnotationKey,
						},
					},
				},
			},
		}
		return &central.MsgFromSensor{Msg: &msg}, nil
	case deduper.TypeComplianceOperatorScan.String():
		msg := central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     resID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorScan{
					ComplianceOperatorScan: &storage.ComplianceOperatorScan{Id: resID},
				},
			},
		}
		return &central.MsgFromSensor{Msg: &msg}, nil
	case deduper.TypeComplianceOperatorScanSettingBinding.String():
		msg := central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     resID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorScanSettingBinding{
					ComplianceOperatorScanSettingBinding: &storage.ComplianceOperatorScanSettingBinding{Id: resID},
				},
			},
		}
		return &central.MsgFromSensor{Msg: &msg}, nil
	default:
		return nil, errors.Errorf("Not implemented for resource type %v", resType)
	}
}
