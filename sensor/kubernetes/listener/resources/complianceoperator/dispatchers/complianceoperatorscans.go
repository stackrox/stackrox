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

// ScanDispatcher handles compliance operator scan objects
type ScanDispatcher struct {
	reconciliationStore reconciliation.Store
}

// NewScanDispatcher creates and returns a new scan dispatcher
func NewScanDispatcher(store reconciliation.Store) *ScanDispatcher {
	return &ScanDispatcher{
		reconciliationStore: store,
	}
}

// ProcessEvent processes a compliance operator scan
func (c *ScanDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	var complianceScan v1alpha1.ComplianceScan

	unstructuredObject, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Not of type 'unstructured': %T", obj)
		return nil
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.Object, &complianceScan); err != nil {
		log.Errorf("error converting unstructured to compliance scan: %v", err)
		return nil
	}

	protoScan := &storage.ComplianceOperatorScan{
		Id:          string(complianceScan.UID),
		Name:        complianceScan.Name,
		ProfileId:   complianceScan.Spec.Profile,
		Labels:      complianceScan.Labels,
		Annotations: complianceScan.Annotations,
	}
	events := []*central.SensorEvent{
		{
			Id:     protoScan.GetId(),
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorScan{
				ComplianceOperatorScan: protoScan,
			},
		},
	}
	if action == central.ResourceAction_REMOVE_RESOURCE {
		c.reconciliationStore.Remove(deduper.TypeComplianceOperatorScan.String(), string(complianceScan.GetUID()))
	} else {
		c.reconciliationStore.Upsert(deduper.TypeComplianceOperatorScan.String(), string(complianceScan.GetUID()))
	}
	return component.NewEvent(events...)
}
