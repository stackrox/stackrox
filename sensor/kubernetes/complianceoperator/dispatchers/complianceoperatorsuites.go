package dispatchers

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// SuitesDispatcher handles compliance operator suites
type SuitesDispatcher struct{}

// NewSuitesDispatcher creates and returns a new compliance suite dispatcher.
func NewSuitesDispatcher() *SuitesDispatcher {
	return &SuitesDispatcher{}
}

// ProcessEvent processes a suite event
func (c *SuitesDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	// compliance operator suites are only processed for compliance V2.
	if !centralcaps.Has(centralsensor.ComplianceV2Integrations) {
		return nil
	}

	unstructuredObject, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Not of type 'unstructured': %T", obj)
		return nil
	}

	uid := string(unstructuredObject.GetUID())

	status, _ := unstructuredObject.Object["status"].(map[string]interface{})
	statusPhase, _ := status["phase"].(string)
	statusResult, _ := status["result"].(string)
	statusErrorMsg, _ := status["errorMessage"].(string)
	conditionsList, _, _ := unstructured.NestedSlice(unstructuredObject.Object, "status", "conditions")

	events := []*central.SensorEvent{
		{
			Id:     uid,
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorSuiteV2{
				ComplianceOperatorSuiteV2: &central.ComplianceOperatorSuiteV2{
					Id:   uid,
					Name: unstructuredObject.GetName(),
					Status: &central.ComplianceOperatorStatus{
						Phase:        statusPhase,
						Result:       statusResult,
						ErrorMessage: statusErrorMsg,
						Conditions:   getStatusConditions(conditionsList),
					},
				},
			},
		},
	}

	return component.NewEvent(events...)
}
