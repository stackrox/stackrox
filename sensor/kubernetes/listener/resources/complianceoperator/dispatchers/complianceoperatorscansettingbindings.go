package dispatchers

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/complianceoperator/api/v1alpha1"
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

// ProcessEvent processes a scan setting binding event
func (c *ScanSettingBindings) ProcessEvent(obj, _ interface{}, action central.ResourceAction) []*central.SensorEvent {
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

	profiles := make([]*storage.ComplianceOperatorScanSettingBinding_Profile, 0, len(scanSettingBindings.Profiles))
	for _, p := range scanSettingBindings.Profiles {
		profiles = append(profiles, &storage.ComplianceOperatorScanSettingBinding_Profile{
			Name: p.Name,
		})
	}

	id := string(scanSettingBindings.UID)
	return []*central.SensorEvent{
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
}
