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

// ActivityClassificationSortOrder returns the sort bucket for observed runtime activity.
// Lower values are more urgent. Unknown is intentionally sorted last for the default ascending sort.
func ActivityClassificationSortOrder(classification storage.RiskActivityClassification) int32 {
	switch classification {
	case storage.RiskActivityClassification_RISK_ACTIVITY_RESPOND_NOW:
		return 1
	case storage.RiskActivityClassification_RISK_ACTIVITY_INVESTIGATE_NOW:
		return 2
	case storage.RiskActivityClassification_RISK_ACTIVITY_NONE_OBSERVED:
		return 3
	default:
		return unknownRiskPrioritySortOrder
	}
}

// ExposureClassificationSortOrder returns the sort bucket for unrealized exposure/blast-radius risk.
// Lower values are more urgent. Unknown is intentionally sorted last for the default ascending sort.
func ExposureClassificationSortOrder(classification storage.RiskExposureClassification) int32 {
	switch classification {
	case storage.RiskExposureClassification_RISK_EXPOSURE_REMEDIATE_NOW:
		return 1
	case storage.RiskExposureClassification_RISK_EXPOSURE_REMEDIATE:
		return 2
	case storage.RiskExposureClassification_RISK_EXPOSURE_NONE_DETECTED:
		return 3
	default:
		return unknownRiskPrioritySortOrder
	}
}

// PostureClassificationSortOrder returns the sort bucket for posture debt.
// Lower values indicate more posture debt. Unknown is intentionally sorted last for the default ascending sort.
func PostureClassificationSortOrder(classification storage.RiskPostureClassification) int32 {
	switch classification {
	case storage.RiskPostureClassification_RISK_POSTURE_CRITICAL:
		return 1
	case storage.RiskPostureClassification_RISK_POSTURE_HIGH:
		return 2
	case storage.RiskPostureClassification_RISK_POSTURE_MEDIUM:
		return 3
	case storage.RiskPostureClassification_RISK_POSTURE_LOW:
		return 4
	case storage.RiskPostureClassification_RISK_POSTURE_NONE_DETECTED:
		return 5
	default:
		return unknownRiskPrioritySortOrder
	}
}
