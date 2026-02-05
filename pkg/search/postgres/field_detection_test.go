package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldNameToDBAlias(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		expected  string
	}{
		{
			name:      "single word lowercase",
			fieldName: "name",
			expected:  "name",
		},
		{
			name:      "single word uppercase",
			fieldName: "NAME",
			expected:  "name",
		},
		{
			name:      "two words with space",
			fieldName: "Secret Type",
			expected:  "secret_type",
		},
		{
			name:      "multiple words",
			fieldName: "Cluster Resource Name",
			expected:  "cluster_resource_name",
		},
		{
			name:      "already lowercase with underscores (should still work)",
			fieldName: "secret_type",
			expected:  "secret_type",
		},
		{
			name:      "mixed case with spaces",
			fieldName: "Created Time",
			expected:  "created_time",
		},
		{
			name:      "camelCase gets split on space boundaries",
			fieldName: "secretType",
			expected:  "secrettype",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FieldNameToDBAlias(tt.fieldName)
			assert.Equal(t, tt.expected, result, "FieldNameToDBAlias(%q) should return %q but got %q", tt.fieldName, tt.expected, result)
		})
	}
}
