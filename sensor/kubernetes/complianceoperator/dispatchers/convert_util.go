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

// populateCelFieldsFromUnstructured extracts CEL rule fields from the raw
// unstructured object map. This is used for regular Rule CRDs whose Go struct
// (v1.8.2) does not have CEL fields, but the K8s object may carry them when
// the cluster runs CO >= 1.9.0.
func populateCelFieldsFromUnstructured(rule *central.ComplianceOperatorRuleV2, obj map[string]interface{}) {
	spec, ok := obj["spec"].(map[string]interface{})
	if !ok {
		return
	}

	if st, ok := spec["scannerType"].(string); ok {
		rule.ScannerType = st
	}
	if expr, ok := spec["expression"].(string); ok {
		rule.Expression = expr
	}
	if fr, ok := spec["failureReason"].(string); ok {
		rule.FailureReason = fr
	}

	inputsList, ok := spec["inputs"].([]interface{})
	if !ok {
		return
	}
	inputs := make([]*central.ComplianceOperatorCelInput, 0, len(inputsList))
	for _, item := range inputsList {
		inputMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		celInput := &central.ComplianceOperatorCelInput{}
		if name, ok := inputMap["name"].(string); ok {
			celInput.Name = name
		}
		if k8sSpec, ok := inputMap["kubernetesInputSpec"].(map[string]interface{}); ok {
			if g, ok := k8sSpec["group"].(string); ok {
				celInput.ApiGroup = g
			}
			if v, ok := k8sSpec["apiVersion"].(string); ok {
				celInput.ApiVersion = v
			}
			if r, ok := k8sSpec["resource"].(string); ok {
				celInput.Resource = r
			}
			if ns, ok := k8sSpec["resourceNamespace"].(string); ok {
				celInput.ResourceNamespace = ns
			}
			if rn, ok := k8sSpec["resourceName"].(string); ok {
				celInput.ResourceName = rn
			}
		}
		inputs = append(inputs, celInput)
	}
	rule.Inputs = inputs
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
