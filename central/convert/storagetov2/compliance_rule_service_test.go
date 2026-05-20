package storagetov2

import (
	"testing"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
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
