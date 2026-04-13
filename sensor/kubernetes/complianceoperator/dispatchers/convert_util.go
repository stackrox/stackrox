package dispatchers

import (
	"github.com/stackrox/rox/generated/internalapi/central"
)

func severityToV2Severity(severity string) central.ComplianceOperatorRuleSeverity {
	switch severity {
	case checkResultSeverityHigh:
		return central.ComplianceOperatorRuleSeverity_HIGH_RULE_SEVERITY
	case checkResultSeverityMedium:
		return central.ComplianceOperatorRuleSeverity_MEDIUM_RULE_SEVERITY
	case checkResultSeverityLow:
		return central.ComplianceOperatorRuleSeverity_LOW_RULE_SEVERITY
	case checkResultSeverityInfo:
		return central.ComplianceOperatorRuleSeverity_INFO_RULE_SEVERITY
	case checkResultSeverityUnknown:
		return central.ComplianceOperatorRuleSeverity_UNKNOWN_RULE_SEVERITY
	default:
		return central.ComplianceOperatorRuleSeverity_UNSET_RULE_SEVERITY
	}
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
