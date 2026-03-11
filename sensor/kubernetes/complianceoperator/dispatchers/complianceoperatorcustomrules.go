package dispatchers

import (
	"strings"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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

	customRule := &v1alpha1.CustomRule{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.Object, customRule); err != nil {
		log.Errorf("error converting unstructured to compliance custom rule: %v", err)
		return nil
	}

	// This ID is used to tell us which clusters have which custom rules. It is also useful for the deduping from sensor.
	id := string(customRule.UID)

	fixes := make([]*central.ComplianceOperatorRuleV2_Fix, 0, len(customRule.Spec.AvailableFixes))
	for _, r := range customRule.Spec.AvailableFixes {
		fixes = append(fixes, &central.ComplianceOperatorRuleV2_Fix{
			Platform:   r.Platform,
			Disruption: r.Disruption,
		})
	}

	// CustomRule objects don't have the compliance.openshift.io/rule annotation that regular Rule
	// objects get from the profile parser. The compliance operator sets that annotation on check
	// results using IDToDNSFriendlyName(Spec.ID), which replaces underscores with hyphens. We
	// synthesize the same value here so that BuildNameRefID produces matching RuleRefIds on both
	// the rule and the check result sides.
	annotations := make(map[string]string, len(customRule.Annotations)+1)
	for k, v := range customRule.Annotations {
		annotations[k] = v
	}
	annotations[v1alpha1.RuleIDAnnotationKey] = strings.ToLower(strings.ReplaceAll(customRule.Spec.ID, "_", "-"))

	events := []*central.SensorEvent{
		{
			Id:     id,
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorRuleV2{
				ComplianceOperatorRuleV2: &central.ComplianceOperatorRuleV2{
					RuleId:       customRule.Spec.ID,
					Id:           id,
					Name:         customRule.Name,
					RuleType:     customRule.Spec.CheckType,
					Severity:     ruleSeverityToV2Severity(customRule.Spec.Severity),
					Labels:       customRule.Labels,
					Annotations:  annotations,
					Title:        customRule.Spec.Title,
					Description:  customRule.Spec.Description,
					Rationale:    customRule.Spec.Rationale,
					Fixes:        fixes,
					Warning:      customRule.Spec.Warning,
					Instructions: customRule.Spec.Instructions,
					OperatorKind: central.ComplianceOperatorRuleV2_CUSTOM_RULE,
				},
			},
		},
	}

	return component.NewEvent(events...)
}
