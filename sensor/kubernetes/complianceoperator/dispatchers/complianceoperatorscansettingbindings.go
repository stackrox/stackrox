package dispatchers

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// ScanSettingBindings handles compliance operator scan setting bindings
type ScanSettingBindings struct {
}

// NewScanSettingBindingsDispatcher creates and returns a new scan setting binding dispatcher
func NewScanSettingBindingsDispatcher() *ScanSettingBindings {
	return &ScanSettingBindings{}
}

func getProfileNames(profiles []v1alpha1.NamedObjectReference) []string {
	profileNames := make([]string, 0, len(profiles))
	for _, p := range profiles {
		profileNames = append(profileNames, p.Name)
	}
	return profileNames
}

func getStatusConditions(conditions v1alpha1.Conditions) []*central.ComplianceOperatorCondition {
	statusConditions := make([]*central.ComplianceOperatorCondition, 0, len(conditions))
	for _, c := range conditions {
		lastTransitionTime, err := protocompat.ConvertTimeToTimestampOrError(c.LastTransitionTime.Time)
		if err != nil {
			log.Warnf("unable to convert last transition time %v, skipping condition", err)
			continue
		}
		statusConditions = append(statusConditions, &central.ComplianceOperatorCondition{
			Type:               string(c.Type),
			Status:             string(c.Status),
			Reason:             string(c.Reason),
			Message:            c.Message,
			LastTransitionTime: lastTransitionTime,
		})
	}
	return statusConditions
}

// ProcessEvent processes a scan setting binding event
func (c *ScanSettingBindings) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	var scanSettingBindings v1alpha1.ScanSettingBinding

	unstructuredObject, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Not of type 'unstructured': %T", obj)
		return nil
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.Object, &scanSettingBindings); err != nil {
		log.Errorf("error converting unstructured to compliance scan setting binding result: %v", err)
		return nil
	}
	id := string(scanSettingBindings.UID)

	profiles := make([]*storage.ComplianceOperatorScanSettingBinding_Profile, 0, len(scanSettingBindings.Profiles))
	for _, p := range scanSettingBindings.Profiles {
		profiles = append(profiles, &storage.ComplianceOperatorScanSettingBinding_Profile{
			Name: p.Name,
		})
	}

	events := []*central.SensorEvent{
		{
			Id:     id,
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorScanSettingBinding{
				ComplianceOperatorScanSettingBinding: &storage.ComplianceOperatorScanSettingBinding{
					Id:          id,
					Name:        scanSettingBindings.Name,
					Labels:      scanSettingBindings.Labels,
					Annotations: scanSettingBindings.Annotations,
					Profiles:    profiles,
				},
			},
		},
	}

	if centralcaps.Has(centralsensor.ComplianceV2Integrations) {
		events = append(events, &central.SensorEvent{
			Id:     id,
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorScanSettingBindingV2{
				ComplianceOperatorScanSettingBindingV2: &central.ComplianceOperatorScanSettingBindingV2{
					Id:           id,
					Name:         scanSettingBindings.Name,
					ProfileNames: getProfileNames(scanSettingBindings.Profiles),
					Status: &central.ComplianceOperatorStatus{
						Conditions: getStatusConditions(scanSettingBindings.Status.Conditions),
					},
					ScanSettingName: scanSettingBindings.SettingsRef.Name,
					Labels:          scanSettingBindings.Labels,
					Annotations:     scanSettingBindings.Annotations,
				},
			},
		})
	}

	return component.NewEvent(events...)
}
