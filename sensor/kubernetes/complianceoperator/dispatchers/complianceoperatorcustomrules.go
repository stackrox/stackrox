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
	// CustomRules support is tied to TailoredProfiles - centrals that don't support TPs don't support custom rules
	if !centralcaps.Has(centralsensor.ComplianceV2TailoredProfiles) {
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

	inputs := make([]*central.ComplianceOperatorCelInput, 0, len(customRule.Spec.Inputs))
	for _, inp := range customRule.Spec.Inputs {
		inputs = append(inputs, &central.ComplianceOperatorCelInput{
			Name:              inp.Name,
			ApiGroup:          inp.KubernetesInputSpec.Group,
			ApiVersion:        inp.KubernetesInputSpec.APIVersion,
			Resource:          inp.KubernetesInputSpec.Resource,
			ResourceNamespace: inp.KubernetesInputSpec.ResourceNamespace,
			ResourceName:      inp.KubernetesInputSpec.ResourceName,
		})
	}

	events := []*central.SensorEvent{
		{
			Id:     id,
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorRuleV2{
				ComplianceOperatorRuleV2: &central.ComplianceOperatorRuleV2{
					RuleId:        customRule.Spec.ID,
					Id:            id,
					Name:          customRule.Name,
					RuleType:      customRule.Spec.CheckType,
					Severity:      ruleSeverityToV2Severity(customRule.Spec.Severity),
					Labels:        customRule.Labels,
					Annotations:   customRule.Annotations,
					Title:         customRule.Spec.Title,
					Description:   customRule.Spec.Description,
					Rationale:     customRule.Spec.Rationale,
					Fixes:         fixes,
					Warning:       customRule.Spec.Warning,
					Instructions:  customRule.Spec.Instructions,
					OperatorKind:  central.ComplianceOperatorRuleV2_CUSTOM_RULE,
					ScannerType:   string(customRule.Spec.ScannerType),
					Expression:    customRule.Spec.Expression,
					Inputs:        inputs,
					FailureReason: customRule.Spec.FailureReason,
					CustomRuleDetails: &central.ComplianceOperatorRuleV2_CustomRuleDetails{
						Phase:        customRule.Status.Phase,
						ErrorMessage: customRule.Status.ErrorMessage,
					},
				},
			},
		},
	}

	return component.NewEvent(events...)
}
