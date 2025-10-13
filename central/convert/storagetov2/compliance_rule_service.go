package storagetov2

import (
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
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
	}
}
