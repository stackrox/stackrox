package storagetov2

import (
	"testing"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestConvertProfileOperatorKind(t *testing.T) {
	cases := map[string]struct {
		input    storage.ComplianceOperatorProfileV2_OperatorKind
		expected v2.ComplianceProfile_OperatorKind
	}{
		"PROFILE": {
			input:    storage.ComplianceOperatorProfileV2_PROFILE,
			expected: v2.ComplianceProfile_PROFILE,
		},
		"TAILORED_PROFILE": {
			input:    storage.ComplianceOperatorProfileV2_TAILORED_PROFILE,
			expected: v2.ComplianceProfile_TAILORED_PROFILE,
		},
		"UNSPECIFIED is treated as PROFILE": {
			input:    storage.ComplianceOperatorProfileV2_OPERATOR_KIND_UNSPECIFIED,
			expected: v2.ComplianceProfile_PROFILE,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, convertProfileOperatorKind(tc.input))
		})
	}
}

func TestConvertProfileSummaryOperatorKind(t *testing.T) {
	cases := map[string]struct {
		input    storage.ComplianceOperatorProfileV2_OperatorKind
		expected v2.ComplianceProfileSummary_OperatorKind
	}{
		"PROFILE": {
			input:    storage.ComplianceOperatorProfileV2_PROFILE,
			expected: v2.ComplianceProfileSummary_PROFILE,
		},
		"TAILORED_PROFILE": {
			input:    storage.ComplianceOperatorProfileV2_TAILORED_PROFILE,
			expected: v2.ComplianceProfileSummary_TAILORED_PROFILE,
		},
		"UNSPECIFIED is treated as PROFILE": {
			input:    storage.ComplianceOperatorProfileV2_OPERATOR_KIND_UNSPECIFIED,
			expected: v2.ComplianceProfileSummary_PROFILE,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, convertProfileSummaryOperatorKind(tc.input))
		})
	}
}
