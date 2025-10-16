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
		coc := &central.ComplianceOperatorCondition{}
		coc.SetType(string(c.Type))
		coc.SetStatus(string(c.Status))
		coc.SetReason(string(c.Reason))
		coc.SetMessage(c.Message)
		coc.SetLastTransitionTime(lastTransitionTime)
		statusConditions = append(statusConditions, coc)
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
		cp := &storage.ComplianceOperatorScanSettingBinding_Profile{}
		cp.SetName(p.Name)
		profiles = append(profiles, cp)
	}

	events := []*central.SensorEvent{
		central.SensorEvent_builder{
			Id:     id,
			Action: action,
			ComplianceOperatorScanSettingBinding: storage.ComplianceOperatorScanSettingBinding_builder{
				Id:          id,
				Name:        scanSettingBindings.Name,
				Labels:      scanSettingBindings.Labels,
				Annotations: scanSettingBindings.Annotations,
				Profiles:    profiles,
			}.Build(),
		}.Build(),
	}

	if centralcaps.Has(centralsensor.ComplianceV2Integrations) {
		events = append(events, central.SensorEvent_builder{
			Id:     id,
			Action: action,
			ComplianceOperatorScanSettingBindingV2: central.ComplianceOperatorScanSettingBindingV2_builder{
				Id:           id,
				Name:         scanSettingBindings.Name,
				ProfileNames: getProfileNames(scanSettingBindings.Profiles),
				Status: central.ComplianceOperatorStatus_builder{
					Conditions: getStatusConditions(scanSettingBindings.Status.Conditions),
				}.Build(),
				ScanSettingName: scanSettingBindings.SettingsRef.Name,
				Labels:          scanSettingBindings.Labels,
				Annotations:     scanSettingBindings.Annotations,
			}.Build(),
		}.Build())
	}

	return component.NewEvent(events...)
}
