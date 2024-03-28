package storagetov2

import (
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
)

// ComplianceRule converts summary object to V2 API summary object
func ComplianceRule(incoming *storage.ComplianceOperatorRuleV2) *v2.ComplianceRule {
	return &v2.ComplianceRule{
		Name:        incoming.GetName(),
		RuleVersion: "",
		RuleType:    incoming.GetRuleType(),
		Severity:    incoming.GetSeverity().String(),
		Standard:    incoming.Get,
		Control:     incoming.GetControls(),
		Title:       incoming.GetTitle(),
		Description: incoming.GetDescription(),
		Rationale:   incoming.GetRationale(),
		Fixes:       incoming.GetFixes(),
		Id:          incoming.GetId(),
		RuleId:      incoming.GetRuleId(),
	}
}
