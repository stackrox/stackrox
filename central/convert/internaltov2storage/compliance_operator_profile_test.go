package internaltov2storage

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

func TestStorageToCentralProfileKind(t *testing.T) {
	testCases := []struct {
		name     string
		input    storage.ComplianceOperatorProfileV2_OperatorKind
		expected central.ComplianceOperatorProfileV2_OperatorKind
	}{
		{
			name:     "profile",
			input:    storage.ComplianceOperatorProfileV2_PROFILE,
			expected: central.ComplianceOperatorProfileV2_PROFILE,
		},
		{
			name:     "tailored profile",
			input:    storage.ComplianceOperatorProfileV2_TAILORED_PROFILE,
			expected: central.ComplianceOperatorProfileV2_TAILORED_PROFILE,
		},
		{
			name:     "unspecified",
			input:    storage.ComplianceOperatorProfileV2_OPERATOR_KIND_UNSPECIFIED,
			expected: central.ComplianceOperatorProfileV2_OPERATOR_KIND_UNSPECIFIED,
		},
		{
			name:     "unknown kind falls back to unspecified",
			input:    storage.ComplianceOperatorProfileV2_OperatorKind(999),
			expected: central.ComplianceOperatorProfileV2_OPERATOR_KIND_UNSPECIFIED,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := StorageToCentralProfileKind(tc.input)
			if actual != tc.expected {
				t.Fatalf("unexpected mapping: got %v want %v", actual, tc.expected)
			}
		})
	}
}
