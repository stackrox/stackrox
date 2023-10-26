package dispatchers

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common/deduper"
	"github.com/stackrox/rox/sensor/common/store/reconciliation"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// ScanSettingBindings handles compliance operator scan setting bindings
type ScanSettingBindings struct {
	reconciliationStore reconciliation.Store
}

// NewScanSettingBindingsDispatcher creates and returns a new scan setting binding dispatcher
func NewScanSettingBindingsDispatcher(store reconciliation.Store) *ScanSettingBindings {
	return &ScanSettingBindings{
		reconciliationStore: store,
	}
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

	profiles := make([]*storage.ComplianceOperatorScanSettingBinding_Profile, 0, len(scanSettingBindings.Profiles))
	for _, p := range scanSettingBindings.Profiles {
		profiles = append(profiles, &storage.ComplianceOperatorScanSettingBinding_Profile{
			Name: p.Name,
		})
	}

	id := string(scanSettingBindings.UID)
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
	if action == central.ResourceAction_REMOVE_RESOURCE {
		c.reconciliationStore.Remove(deduper.TypeComplianceOperatorScanSettingBinding.String(), id)
	} else {
		c.reconciliationStore.Upsert(deduper.TypeComplianceOperatorScanSettingBinding.String(), id)
	}
	return component.NewEvent(events...)
}
