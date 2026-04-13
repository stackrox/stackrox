package dispatchers

import (
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ScanSettingBindings handles compliance operator scan setting bindings
type ScanSettingBindings struct {
}

// NewScanSettingBindingsDispatcher creates and returns a new scan setting binding dispatcher
func NewScanSettingBindingsDispatcher() *ScanSettingBindings {
	return &ScanSettingBindings{}
}

func getProfileNames(profiles []interface{}) []string {
	profileNames := make([]string, 0, len(profiles))
	for _, p := range profiles {
		if profileMap, ok := p.(map[string]interface{}); ok {
			if name, ok := profileMap["name"].(string); ok {
				profileNames = append(profileNames, name)
			}
		}
	}
	return profileNames
}

func getStatusConditions(conditionsList []interface{}) []*central.ComplianceOperatorCondition {
	statusConditions := make([]*central.ComplianceOperatorCondition, 0, len(conditionsList))
	for _, c := range conditionsList {
		condMap, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		condType, _ := condMap["type"].(string)
		condStatus, _ := condMap["status"].(string)
		condReason, _ := condMap["reason"].(string)
		condMessage, _ := condMap["message"].(string)
		lastTransitionTimeStr, _ := condMap["lastTransitionTime"].(string)

		if lastTransitionTimeStr == "" {
			log.Warnf("unable to convert last transition time (empty), skipping condition")
			continue
		}
		lastTransitionTime, err := time.Parse(time.RFC3339, lastTransitionTimeStr)
		if err != nil {
			log.Warnf("unable to parse last transition time %q, skipping condition: %v", lastTransitionTimeStr, err)
			continue
		}
		lastTransitionTimestamp, err := protocompat.ConvertTimeToTimestampOrError(lastTransitionTime)
		if err != nil {
			log.Warnf("unable to convert last transition time %v, skipping condition", err)
			continue
		}

		statusConditions = append(statusConditions, &central.ComplianceOperatorCondition{
			Type:               condType,
			Status:             condStatus,
			Reason:             condReason,
			Message:            condMessage,
			LastTransitionTime: lastTransitionTimestamp,
		})
	}
	return statusConditions
}

// ProcessEvent processes a scan setting binding event
func (c *ScanSettingBindings) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	unstructuredObject, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Not of type 'unstructured': %T", obj)
		return nil
	}

	id := string(unstructuredObject.GetUID())

	profilesList, _, _ := unstructured.NestedSlice(unstructuredObject.Object, "profiles")
	profiles := make([]*storage.ComplianceOperatorScanSettingBinding_Profile, 0, len(profilesList))
	for _, p := range profilesList {
		if profileMap, ok := p.(map[string]interface{}); ok {
			if name, ok := profileMap["name"].(string); ok {
				profiles = append(profiles, &storage.ComplianceOperatorScanSettingBinding_Profile{
					Name: name,
				})
			}
		}
	}

	events := []*central.SensorEvent{
		{
			Id:     id,
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorScanSettingBinding{
				ComplianceOperatorScanSettingBinding: &storage.ComplianceOperatorScanSettingBinding{
					Id:          id,
					Name:        unstructuredObject.GetName(),
					Labels:      unstructuredObject.GetLabels(),
					Annotations: unstructuredObject.GetAnnotations(),
					Profiles:    profiles,
				},
			},
		},
	}

	if centralcaps.Has(centralsensor.ComplianceV2Integrations) {
		conditionsList, _, _ := unstructured.NestedSlice(unstructuredObject.Object, "status", "conditions")
		settingsRefName, _, _ := unstructured.NestedString(unstructuredObject.Object, "settingsRef", "name")

		events = append(events, &central.SensorEvent{
			Id:     id,
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorScanSettingBindingV2{
				ComplianceOperatorScanSettingBindingV2: &central.ComplianceOperatorScanSettingBindingV2{
					Id:           id,
					Name:         unstructuredObject.GetName(),
					ProfileNames: getProfileNames(profilesList),
					Status: &central.ComplianceOperatorStatus{
						Conditions: getStatusConditions(conditionsList),
					},
					ScanSettingName: settingsRefName,
					Labels:          unstructuredObject.GetLabels(),
					Annotations:     unstructuredObject.GetAnnotations(),
				},
			},
		})
	}

	return component.NewEvent(events...)
}
