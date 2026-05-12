package storagetov2

import (
	"github.com/pkg/errors"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
)

// ComplianceRule converts summary object to V2 API summary object
func ComplianceRule(incoming *storage.ComplianceOperatorRuleV2) *v2.ComplianceRule {
	fixes := make([]*v2.ComplianceRule_Fix, 0, len(incoming.GetFixes()))
	for _, fix := range incoming.GetFixes() {
		fixes = append(fixes, &v2.ComplianceRule_Fix{
			Platform:   fix.GetPlatform(),
			Disruption: fix.GetDisruption(),
		})
	}

	return &v2.ComplianceRule{
		Name:         incoming.GetName(),
		RuleType:     incoming.GetRuleType(),
		Severity:     incoming.GetSeverity().String(),
		Title:        incoming.GetTitle(),
		Description:  incoming.GetDescription(),
		Rationale:    incoming.GetRationale(),
		Fixes:        fixes,
		Id:           incoming.GetId(),
		RuleId:       incoming.GetRuleId(),
		Instructions: incoming.GetInstructions(),
		Warning:      incoming.GetWarning(),
		ParentRule:   incoming.GetParentRule(),
		OperatorKind: convertRuleOperatorKind(incoming.GetOperatorKind()),
	}
}

func convertRuleOperatorKind(kind storage.ComplianceOperatorRuleV2_OperatorKind) v2.ComplianceRule_OperatorKind {
	switch kind {
	case storage.ComplianceOperatorRuleV2_RULE:
		return v2.ComplianceRule_RULE
	case storage.ComplianceOperatorRuleV2_CUSTOM_RULE:
		return v2.ComplianceRule_CUSTOM_RULE
	case storage.ComplianceOperatorRuleV2_OPERATOR_KIND_UNSPECIFIED:
		// Older sensors do not set OperatorKind for regular (non-custom) rules,
		// so UNSPECIFIED is treated as RULE. This fallback can be removed when
		// versions that don't set OperatorKind (<= 4.10) are not supported.
		return v2.ComplianceRule_RULE
	default:
		utils.Should(errors.Errorf("unhandled rule operator kind %s", kind))
		return v2.ComplianceRule_OPERATOR_KIND_UNSPECIFIED
	}
}
