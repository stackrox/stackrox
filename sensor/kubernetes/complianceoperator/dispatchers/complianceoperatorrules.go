package dispatchers

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// RulesDispatcher handles compliance operator rules
type RulesDispatcher struct{}

// NewRulesDispatcher creates and returns a new compliance rule dispatcher.
func NewRulesDispatcher() *RulesDispatcher {
	return &RulesDispatcher{}
}

// ProcessEvent processes a rule event
func (c *RulesDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	unstructuredObject, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Not of type 'unstructured': %T", obj)
		return nil
	}

	id := string(unstructuredObject.GetUID())

	// Rule fields are inline (RulePayload is embedded with json:",inline"),
	// so they appear at the top level of the object, not under "spec".
	ruleID, _, _ := unstructured.NestedString(unstructuredObject.Object, "id")
	title, _, _ := unstructured.NestedString(unstructuredObject.Object, "title")
	description, _, _ := unstructured.NestedString(unstructuredObject.Object, "description")
	rationale, _, _ := unstructured.NestedString(unstructuredObject.Object, "rationale")
	warning, _, _ := unstructured.NestedString(unstructuredObject.Object, "warning")
	severity, _, _ := unstructured.NestedString(unstructuredObject.Object, "severity")
	checkType, _, _ := unstructured.NestedString(unstructuredObject.Object, "checkType")
	instructions, _, _ := unstructured.NestedString(unstructuredObject.Object, "instructions")

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
					RuleId:      ruleID,
					Name:        unstructuredObject.GetName(),
					Title:       title,
					Labels:      unstructuredObject.GetLabels(),
					Annotations: unstructuredObject.GetAnnotations(),
					Description: description,
					Rationale:   rationale,
				},
			},
		},
	}

	if centralcaps.Has(centralsensor.ComplianceV2Integrations) {
		availableFixesList, _, _ := unstructured.NestedSlice(unstructuredObject.Object, "availableFixes")
		fixes := make([]*central.ComplianceOperatorRuleV2_Fix, 0, len(availableFixesList))
		for _, f := range availableFixesList {
			if fixMap, ok := f.(map[string]interface{}); ok {
				platform, _ := fixMap["platform"].(string)
				disruption, _ := fixMap["disruption"].(string)
				fixes = append(fixes, &central.ComplianceOperatorRuleV2_Fix{
					Platform:   platform,
					Disruption: disruption,
				})
			}
		}

		events = append(events, &central.SensorEvent{
			Id:     id,
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorRuleV2{
				ComplianceOperatorRuleV2: &central.ComplianceOperatorRuleV2{
					RuleId:       ruleID,
					Id:           id,
					Name:         unstructuredObject.GetName(),
					RuleType:     checkType,
					Severity:     ruleSeverityToV2Severity(severity),
					Labels:       unstructuredObject.GetLabels(),
					Annotations:  unstructuredObject.GetAnnotations(),
					Title:        title,
					Description:  description,
					Rationale:    rationale,
					Fixes:        fixes,
					Warning:      warning,
					Instructions: instructions,
				},
			},
		})
	}

	return component.NewEvent(events...)
}
