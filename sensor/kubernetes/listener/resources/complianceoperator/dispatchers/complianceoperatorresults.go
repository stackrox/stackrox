package dispatchers

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// ResultDispatcher handles compliance check result objects
type ResultDispatcher struct {
}

// NewResultDispatcher creates and returns a new compliance check result dispatcher.
func NewResultDispatcher() *ResultDispatcher {
	return &ResultDispatcher{}
}

func statusToProtoStatus(status v1alpha1.ComplianceCheckStatus) storage.ComplianceOperatorCheckResult_CheckStatus {
	switch status {
	case v1alpha1.CheckResultPass:
		return storage.ComplianceOperatorCheckResult_PASS
	case v1alpha1.CheckResultFail:
		return storage.ComplianceOperatorCheckResult_FAIL
	case v1alpha1.CheckResultInfo:
		return storage.ComplianceOperatorCheckResult_INFO
	case v1alpha1.CheckResultManual:
		return storage.ComplianceOperatorCheckResult_MANUAL
	case v1alpha1.CheckResultError:
		return storage.ComplianceOperatorCheckResult_ERROR
	case v1alpha1.CheckResultNotApplicable:
		return storage.ComplianceOperatorCheckResult_NOT_APPLICABLE
	case v1alpha1.CheckResultInconsistent:
		return storage.ComplianceOperatorCheckResult_INCONSISTENT
	default:
		return storage.ComplianceOperatorCheckResult_UNSET
	}
}

// ProcessEvent processes a compliance operator check result
func (c *ResultDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	var complianceCheckResult v1alpha1.ComplianceCheckResult

	unstructuredObject, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Not of type 'unstructured': %T", obj)
		return nil
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.Object, &complianceCheckResult); err != nil {
		log.Errorf("error converting unstructured to compliance check result: %v", err)
		return nil
	}

	id := string(complianceCheckResult.UID)
	events := []*central.SensorEvent{
		{
			Id:     id,
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorResult{
				ComplianceOperatorResult: &storage.ComplianceOperatorCheckResult{
					Id:           id,
					CheckId:      complianceCheckResult.ID,
					CheckName:    complianceCheckResult.Name,
					Status:       statusToProtoStatus(complianceCheckResult.Status),
					Description:  complianceCheckResult.Description,
					Instructions: complianceCheckResult.Instructions,
					Labels:       complianceCheckResult.Labels,
					Annotations:  complianceCheckResult.Annotations,
				},
			},
		},
	}
	return component.NewEvent(events...)
}
