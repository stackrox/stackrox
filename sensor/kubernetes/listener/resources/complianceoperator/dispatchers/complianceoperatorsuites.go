package dispatchers

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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
	var complianceSuite v1alpha1.ComplianceSuite

	unstructuredObject, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Not of type 'unstructured': %T", obj)
		return nil
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.Object, &complianceSuite); err != nil {
		log.Errorf("error converting unstructured to compliance suite: %v", err)
		return nil
	}

	events := []*central.SensorEvent{
		{
			Id:     string(complianceSuite.GetUID()),
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorSuiteV2{
				ComplianceOperatorSuiteV2: &central.ComplianceOperatorSuiteV2{
					Id:   string(complianceSuite.GetUID()),
					Name: complianceSuite.Name,
					Status: &central.ComplianceOperatorSuiteV2_Status{
						Phase:        string(complianceSuite.Status.Phase),
						Result:       string(complianceSuite.Status.Result),
						ErrorMessage: string(complianceSuite.Status.ErrorMessage),
						Conditions:   getStatusConditions(complianceSuite.Status.Conditions),
					},
				},
			},
		},
	}

	return component.NewEvent(events...)
}
