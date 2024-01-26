package dispatchers

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// RulesDispatcher handles compliance operator rules
type RulesDispatcher struct{}

// NewRulesDispatcher creates and returns a new compliance rule dispatcher.
func NewRulesDispatcher() *RulesDispatcher {
	return &RulesDispatcher{}
}

// ProcessEvent processes a rule event
func (c *RulesDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	var complianceRule v1alpha1.Rule

	unstructuredObject, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Not of type 'unstructured': %T", obj)
		return nil
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.Object, &complianceRule); err != nil {
		log.Errorf("error converting unstructured to compliance rule: %v", err)
		return nil
	}
	id := string(complianceRule.UID)
	events := []*central.SensorEvent{
		{
			Id:     id,
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorRule{
				ComplianceOperatorRule: &storage.ComplianceOperatorRule{
					Id:          id,
					RuleId:      complianceRule.ID,
					Name:        complianceRule.Name,
					Title:       complianceRule.Title,
					Labels:      complianceRule.Labels,
					Annotations: complianceRule.Annotations,
					Description: complianceRule.Description,
					Rationale:   complianceRule.Rationale,
				},
			},
		},
	}
	return component.NewEvent(events...)
}
