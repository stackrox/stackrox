package storagetov2

import (
	"testing"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertRuleOperatorKind(t *testing.T) {
	cases := map[string]struct {
		input    storage.ComplianceOperatorRuleV2_OperatorKind
		expected v2.ComplianceRule_OperatorKind
	}{
		"RULE": {
			input:    storage.ComplianceOperatorRuleV2_RULE,
			expected: v2.ComplianceRule_RULE,
		},
		"CUSTOM_RULE": {
			input:    storage.ComplianceOperatorRuleV2_CUSTOM_RULE,
			expected: v2.ComplianceRule_CUSTOM_RULE,
		},
		"UNSPECIFIED is treated as RULE": {
			input:    storage.ComplianceOperatorRuleV2_OPERATOR_KIND_UNSPECIFIED,
			expected: v2.ComplianceRule_RULE,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, convertRuleOperatorKind(tc.input))
		})
	}
}

func TestComplianceRule_CelFieldsConversion(t *testing.T) {
	incoming := &storage.ComplianceOperatorRuleV2{
		Id:            "rule-uid",
		Name:          "check-cm",
		RuleType:      "Platform",
		Severity:      storage.RuleSeverity_HIGH_RULE_SEVERITY,
		OperatorKind:  storage.ComplianceOperatorRuleV2_CUSTOM_RULE,
		ScannerType:   "CEL",
		Expression:    `input.cm.data["key"] == "val"`,
		FailureReason: "key not found",
		Inputs: []*storage.ComplianceOperatorCelInput{
			{
				Name:              "cm",
				ApiVersion:        "v1",
				Resource:          "configmaps",
				ResourceNamespace: "default",
				ResourceName:      "my-cm",
			},
		},
		CustomRuleDetails: &storage.ComplianceOperatorRuleV2_CustomRuleDetails{
			Phase:        "Error",
			ErrorMessage: "bad expression",
		},
	}

	result := ComplianceRule(incoming)

	assert.Equal(t, "CEL", result.GetScannerType())
	assert.Equal(t, `input.cm.data["key"] == "val"`, result.GetExpression())
	assert.Equal(t, "key not found", result.GetFailureReason())
	require.Len(t, result.GetInputs(), 1)
	assert.Equal(t, "cm", result.GetInputs()[0].GetName())
	assert.Equal(t, "v1", result.GetInputs()[0].GetApiVersion())
	assert.Equal(t, "configmaps", result.GetInputs()[0].GetResource())
	require.NotNil(t, result.GetCustomRuleDetails())
	assert.Equal(t, "Error", result.GetCustomRuleDetails().GetPhase())
	assert.Equal(t, "bad expression", result.GetCustomRuleDetails().GetErrorMessage())
}

func TestComplianceRule_NoCelFields(t *testing.T) {
	incoming := &storage.ComplianceOperatorRuleV2{
		Id:           "rule-uid",
		Name:         "some-rule",
		OperatorKind: storage.ComplianceOperatorRuleV2_RULE,
	}

	result := ComplianceRule(incoming)

	assert.Empty(t, result.GetScannerType())
	assert.Empty(t, result.GetExpression())
	assert.Empty(t, result.GetInputs())
	assert.Nil(t, result.GetCustomRuleDetails())
}
