package risk

import "github.com/stackrox/rox/generated/storage"

const unknownRiskPrioritySortOrder int32 = 99

// PriorityClassificationSortOrder returns the sort bucket for a risk priority classification.
// Lower values are more urgent. Unknown is intentionally sorted last for the default ascending sort.
func PriorityClassificationSortOrder(classification storage.RiskPriorityClassification) int32 {
	switch classification {
	case storage.RiskPriorityClassification_RISK_PRIORITY_RESPOND_NOW:
		return 1
	case storage.RiskPriorityClassification_RISK_PRIORITY_INVESTIGATE_NOW:
		return 2
	case storage.RiskPriorityClassification_RISK_PRIORITY_REMEDIATE_NOW:
		return 3
	case storage.RiskPriorityClassification_RISK_PRIORITY_REMEDIATE:
		return 4
	case storage.RiskPriorityClassification_RISK_PRIORITY_IMPROVE_POSTURE:
		return 5
	default:
		return unknownRiskPrioritySortOrder
	}
}
