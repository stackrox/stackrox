package manual

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOverrides(t *testing.T) {
	tests := map[string]struct {
		yaml     string
		expected []VulnerabilityOverride
		wantErr  bool
	}{
		"single vulnerability override": {
			yaml: `vulnerabilities:
  - Name: CVE-2023-9999
    Severity: Critical
    NormalizedSeverity: Critical
    FixedInVersion: "1.0.1"
`,
			expected: []VulnerabilityOverride{
				{
					Name:               "CVE-2023-9999",
					Severity:           "Critical",
					NormalizedSeverity: "Critical",
					FixedInVersion:     "1.0.1",
				},
			},
		},
		"multiple vulnerability overrides": {
			yaml: `vulnerabilities:
  - Name: CVE-2023-1111
    Description: First vulnerability
    Severity: High
    NormalizedSeverity: Important
    FixedInVersion: "2.0.0"
    Links: https://example.com/cve-2023-1111
  - Name: CVE-2023-2222
    Description: Second vulnerability
    Severity: Medium
    NormalizedSeverity: Moderate
    FixedInVersion: "3.1.0"
`,
			expected: []VulnerabilityOverride{
				{
					Name:               "CVE-2023-1111",
					Description:        "First vulnerability",
					Severity:           "High",
					NormalizedSeverity: "Important",
					FixedInVersion:     "2.0.0",
					Links:              "https://example.com/cve-2023-1111",
				},
				{
					Name:               "CVE-2023-2222",
					Description:        "Second vulnerability",
					Severity:           "Medium",
					NormalizedSeverity: "Moderate",
					FixedInVersion:     "3.1.0",
				},
			},
		},
		"partial fields": {
			yaml: `vulnerabilities:
  - Name: CVE-2023-3333
    Severity: Low
`,
			expected: []VulnerabilityOverride{
				{
					Name:     "CVE-2023-3333",
					Severity: "Low",
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := ParseOverrides([]byte(tc.yaml))
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseOverrides_Empty(t *testing.T) {
	tests := map[string]struct {
		yaml     string
		expected []VulnerabilityOverride
	}{
		"empty vulnerabilities list": {
			yaml:     `vulnerabilities: []`,
			expected: []VulnerabilityOverride{},
		},
		"no vulnerabilities key": {
			yaml:     ``,
			expected: nil,
		},
		"empty document": {
			yaml:     `{}`,
			expected: nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := ParseOverrides([]byte(tc.yaml))
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseOverrides_InvalidYAML(t *testing.T) {
	tests := map[string]struct {
		yaml string
	}{
		"invalid YAML syntax": {
			yaml: `vulnerabilities:
  - Name: CVE-2023-9999
    Severity: Critical
  invalid line without indentation
`,
		},
		"malformed structure": {
			yaml: `vulnerabilities: "not a list"`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := ParseOverrides([]byte(tc.yaml))
			require.Error(t, err)
		})
	}
}
