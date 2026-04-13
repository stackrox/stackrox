package dispatchers

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// CustomRuleDispatcher handles compliance operator custom rule objects.
// CustomRules are CEL-based compliance checks available in Compliance Operator 1.8.0+.
type CustomRuleDispatcher struct{}

// NewCustomRuleDispatcher creates and returns a new custom rule dispatcher.
func NewCustomRuleDispatcher() *CustomRuleDispatcher {
	return &CustomRuleDispatcher{}
}

// ProcessEvent processes a custom rule event.
// CustomRules are converted to ComplianceOperatorRuleV2 with OperatorKind=CUSTOM_RULE, following the same pattern as TailoredProfiles.
func (c *CustomRuleDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	// CustomRules are only supported by compliance V2
	if !centralcaps.Has(centralsensor.ComplianceV2Integrations) {
		return nil
	}

	unstructuredObject, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Not of type 'unstructured': %T", obj)
		return nil
	}

	// This ID is used to tell us which clusters have which custom rules. It is also useful for the deduping from sensor.
	id := string(unstructuredObject.GetUID())

	spec, _ := unstructuredObject.Object["spec"].(map[string]interface{})

	ruleID, _ := spec["id"].(string)
	checkType, _ := spec["checkType"].(string)
	severity, _ := spec["severity"].(string)
	title, _ := spec["title"].(string)
	description, _ := spec["description"].(string)
	rationale, _ := spec["rationale"].(string)
	warning, _ := spec["warning"].(string)
	instructions, _ := spec["instructions"].(string)

	var fixes []*central.ComplianceOperatorRuleV2_Fix
	if availableFixes, ok := spec["availableFixes"].([]interface{}); ok {
		fixes = make([]*central.ComplianceOperatorRuleV2_Fix, 0, len(availableFixes))
		for _, f := range availableFixes {
			if fixMap, ok := f.(map[string]interface{}); ok {
				platform, _ := fixMap["platform"].(string)
				disruption, _ := fixMap["disruption"].(string)
				fixes = append(fixes, &central.ComplianceOperatorRuleV2_Fix{
					Platform:   platform,
					Disruption: disruption,
				})
			}
		}
	}

	events := []*central.SensorEvent{
		{
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
					OperatorKind: central.ComplianceOperatorRuleV2_CUSTOM_RULE,
				},
			},
		},
	}

	return component.NewEvent(events...)
}
