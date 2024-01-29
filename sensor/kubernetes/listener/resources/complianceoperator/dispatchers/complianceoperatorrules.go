package dispatchers

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/sensor/common/centralcaps"
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
	// We are pulling additional data for rules and using the storage object even in an internal api
	// is a bad practice, so we will make that split now.  V1 and V2 compliance will both need to work for a period
	// of time.  However, we should not need to send the same rule twice, the pipeline can convert the V2 sensor message
	// so V1 and V2 objects can both be stored.

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

	if centralcaps.Has(centralsensor.ComplianceV2Integrations) {
		fixes := make([]*central.ComplianceOperatorRuleV2_Fix, 0, len(complianceRule.AvailableFixes))
		for _, r := range complianceRule.AvailableFixes {
			fixes = append(fixes, &central.ComplianceOperatorRuleV2_Fix{
				Platform:   r.Platform,
				Disruption: r.Disruption,
			})
		}

		events = append(events, &central.SensorEvent{
			Id:     id,
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorRuleV2{
				ComplianceOperatorRuleV2: &central.ComplianceOperatorRuleV2{
					RuleId:      complianceRule.ID,
					Id:          id,
					Name:        complianceRule.Name,
					RuleType:    complianceRule.CheckType,
					Severity:    ruleSeverityToV2Severity(complianceRule.Severity),
					Labels:      complianceRule.Labels,
					Annotations: complianceRule.Annotations,
					Title:       complianceRule.Title,
					Description: complianceRule.Description,
					Rationale:   complianceRule.Rationale,
					Fixes:       fixes,
					Warning:     complianceRule.Warning,
				},
			},
		})
	}

	return component.NewEvent(events...)
}
