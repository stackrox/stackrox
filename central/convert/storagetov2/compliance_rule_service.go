package storagetov2

import (
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
)

// ComplianceRule converts summary object to V2 API summary object
func ComplianceRule(incoming *storage.ComplianceOperatorRuleV2) *v2.ComplianceRule {
	fixes := make([]*v2.ComplianceRule_Fix, 0, len(incoming.GetFixes()))
	for _, fix := range incoming.GetFixes() {
		cf := &v2.ComplianceRule_Fix{}
		cf.SetPlatform(fix.GetPlatform())
		cf.SetDisruption(fix.GetDisruption())
		fixes = append(fixes, cf)
	}

	cr := &v2.ComplianceRule{}
	cr.SetName(incoming.GetName())
	cr.SetRuleType(incoming.GetRuleType())
	cr.SetSeverity(incoming.GetSeverity().String())
	cr.SetTitle(incoming.GetTitle())
	cr.SetDescription(incoming.GetDescription())
	cr.SetRationale(incoming.GetRationale())
	cr.SetFixes(fixes)
	cr.SetId(incoming.GetId())
	cr.SetRuleId(incoming.GetRuleId())
	cr.SetInstructions(incoming.GetInstructions())
	cr.SetWarning(incoming.GetWarning())
	cr.SetParentRule(incoming.GetParentRule())
	return cr
}
