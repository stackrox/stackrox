package utils

import (
	"fmt"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

// CustomRuleEffectiveOperatorKind normalizes operator kind during mixed-version compatibility:
// UNSPECIFIED is treated as RULE.
func CustomRuleEffectiveOperatorKind[T central.ComplianceOperatorRuleV2_OperatorKind | storage.ComplianceOperatorRuleV2_OperatorKind](operatorKind T) T {
	// If kind is already specified (!= 0), return it directly
	if operatorKind != 0 {
		return operatorKind
	}

	// If kind is UNSPECIFIED (0 by convention in both storage and api protos), assume it's a RULE. This is for compatibility
	// with sensors that don't support custom rules and won't fill the OperatorKind field, and must be kept for as long as we
	// support such sensor versions.
	switch any(operatorKind).(type) {
	case central.ComplianceOperatorRuleV2_OperatorKind:
		return T(central.ComplianceOperatorRuleV2_RULE)
	case storage.ComplianceOperatorRuleV2_OperatorKind:
		return T(storage.ComplianceOperatorRuleV2_RULE)
	}

	// The switch covers all possible types in the type parameter constraint
	panic(fmt.Sprintf("unreachable: unexpected type %T", operatorKind))
}
