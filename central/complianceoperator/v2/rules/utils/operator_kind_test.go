package utils

import (
	"testing"

	internalapi "github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestCustomRuleEffectiveOperatorKindInternal(t *testing.T) {
	assert.Equal(t, internalapi.ComplianceOperatorRuleV2_RULE,
		CustomRuleEffectiveOperatorKind(internalapi.ComplianceOperatorRuleV2_OPERATOR_KIND_UNSPECIFIED))
	assert.Equal(t, internalapi.ComplianceOperatorRuleV2_RULE,
		CustomRuleEffectiveOperatorKind(internalapi.ComplianceOperatorRuleV2_RULE))
	assert.Equal(t, internalapi.ComplianceOperatorRuleV2_CUSTOM_RULE,
		CustomRuleEffectiveOperatorKind(internalapi.ComplianceOperatorRuleV2_CUSTOM_RULE))
}

func TestCustomRuleEffectiveOperatorKindStorage(t *testing.T) {
	assert.Equal(t, storage.ComplianceOperatorRuleV2_RULE,
		CustomRuleEffectiveOperatorKind(storage.ComplianceOperatorRuleV2_OPERATOR_KIND_UNSPECIFIED))
	assert.Equal(t, storage.ComplianceOperatorRuleV2_RULE,
		CustomRuleEffectiveOperatorKind(storage.ComplianceOperatorRuleV2_RULE))
	assert.Equal(t, storage.ComplianceOperatorRuleV2_CUSTOM_RULE,
		CustomRuleEffectiveOperatorKind(storage.ComplianceOperatorRuleV2_CUSTOM_RULE))
}
