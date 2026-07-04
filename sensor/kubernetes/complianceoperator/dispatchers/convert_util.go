package dispatchers

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
)

func severityToV2Severity(severity v1alpha1.ComplianceCheckResultSeverity) central.ComplianceOperatorRuleSeverity {
	switch severity {
	case v1alpha1.CheckResultSeverityHigh:
		return central.ComplianceOperatorRuleSeverity_HIGH_RULE_SEVERITY
	case v1alpha1.CheckResultSeverityMedium:
		return central.ComplianceOperatorRuleSeverity_MEDIUM_RULE_SEVERITY
	case v1alpha1.CheckResultSeverityLow:
		return central.ComplianceOperatorRuleSeverity_LOW_RULE_SEVERITY
	case v1alpha1.CheckResultSeverityInfo:
		return central.ComplianceOperatorRuleSeverity_INFO_RULE_SEVERITY
	case v1alpha1.CheckResultSeverityUnknown:
		return central.ComplianceOperatorRuleSeverity_UNKNOWN_RULE_SEVERITY
	default:
		return central.ComplianceOperatorRuleSeverity_UNSET_RULE_SEVERITY
	}
}

func inputPayloadsToCelInputs(payloads []v1alpha1.InputPayload) []*central.ComplianceOperatorCelInput {
	inputs := make([]*central.ComplianceOperatorCelInput, 0, len(payloads))
	for _, inp := range payloads {
		inputs = append(inputs, &central.ComplianceOperatorCelInput{
			Name:              inp.Name,
			ApiGroup:          inp.KubernetesInputSpec.Group,
			ApiVersion:        inp.KubernetesInputSpec.APIVersion,
			Resource:          inp.KubernetesInputSpec.Resource,
			ResourceNamespace: inp.KubernetesInputSpec.ResourceNamespace,
			ResourceName:      inp.KubernetesInputSpec.ResourceName,
		})
	}
	return inputs
}

func ruleSeverityToV2Severity(severity string) central.ComplianceOperatorRuleSeverity {
	switch severity {
	case "high":
		return central.ComplianceOperatorRuleSeverity_HIGH_RULE_SEVERITY
	case "medium":
		return central.ComplianceOperatorRuleSeverity_MEDIUM_RULE_SEVERITY
	case "low":
		return central.ComplianceOperatorRuleSeverity_LOW_RULE_SEVERITY
	case "info":
		return central.ComplianceOperatorRuleSeverity_INFO_RULE_SEVERITY
	case "unknown":
		return central.ComplianceOperatorRuleSeverity_UNKNOWN_RULE_SEVERITY
	default:
		return central.ComplianceOperatorRuleSeverity_UNSET_RULE_SEVERITY
	}
}
