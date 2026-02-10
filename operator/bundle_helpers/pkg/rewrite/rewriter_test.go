package rewrite

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRewriteStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		old      string
		new      string
		expected any
		modified bool
	}{
		{
			name:     "bare string cannot be modified",
			input:    "quay.io/stackrox-io/stackrox-operator:0.0.1",
			old:      "quay.io/stackrox-io/stackrox-operator:0.0.1",
			new:      "quay.io/stackrox-io/stackrox-operator:4.0.0",
			expected: "quay.io/stackrox-io/stackrox-operator:0.0.1",
			modified: false,
		},
		{
			name:     "no match leaves unchanged",
			input:    "some-other-value",
			old:      "not-found",
			new:      "replacement",
			expected: "some-other-value",
			modified: false,
		},
		{
			name: "replace in map values",
			input: map[string]any{
				"containerImage": "quay.io/stackrox-io/stackrox-operator:0.0.1",
				"other":          "unchanged",
			},
			old: "quay.io/stackrox-io/stackrox-operator:0.0.1",
			new: "quay.io/stackrox-io/stackrox-operator:4.0.0",
			expected: map[string]any{
				"containerImage": "quay.io/stackrox-io/stackrox-operator:4.0.0",
				"other":          "unchanged",
			},
			modified: true,
		},
		{
			name: "replace in slice elements",
			input: []any{
				"quay.io/stackrox-io/stackrox-operator:0.0.1",
				"other-value",
			},
			old: "quay.io/stackrox-io/stackrox-operator:0.0.1",
			new: "quay.io/stackrox-io/stackrox-operator:4.0.0",
			expected: []any{
				"quay.io/stackrox-io/stackrox-operator:4.0.0",
				"other-value",
			},
			modified: true,
		},
		{
			name: "replace in nested structures",
			input: map[string]any{
				"outer": map[string]any{
					"inner": []any{
						"quay.io/stackrox-io/stackrox-operator:0.0.1",
					},
				},
			},
			old: "quay.io/stackrox-io/stackrox-operator:0.0.1",
			new: "quay.io/stackrox-io/stackrox-operator:4.0.0",
			expected: map[string]any{
				"outer": map[string]any{
					"inner": []any{
						"quay.io/stackrox-io/stackrox-operator:4.0.0",
					},
				},
			},
			modified: true,
		},
		{
			name: "nested structures traversed but no match",
			input: map[string]any{
				"outer": map[string]any{
					"inner": []any{
						"some-other-value",
						"another-value",
					},
					"field": "unchanged",
				},
				"top": "also-unchanged",
			},
			old: "not-found",
			new: "replacement",
			expected: map[string]any{
				"outer": map[string]any{
					"inner": []any{
						"some-other-value",
						"another-value",
					},
					"field": "unchanged",
				},
				"top": "also-unchanged",
			},
			modified: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modified := RewriteStrings(tt.input, tt.old, tt.new)
			assert.Equal(t, tt.modified, modified)
			assert.Equal(t, tt.expected, tt.input)
		})
	}
}
