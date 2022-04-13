package dispatchers

import (
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/complianceoperator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// ScanDispatcher handles compliance operator scan objects
type ScanDispatcher struct{}

// NewScanDispatcher creates and returns a new scan dispatcher
func NewScanDispatcher() *ScanDispatcher {
	return &ScanDispatcher{}
}

// ProcessEvent processes a compliance operator scan
func (c *ScanDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) []*central.SensorEvent {
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
	return []*central.SensorEvent{
		{
			Id:     protoScan.GetId(),
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorScan{
				ComplianceOperatorScan: protoScan,
			},
		},
	}
}
