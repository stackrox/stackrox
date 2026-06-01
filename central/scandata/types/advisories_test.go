package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAdvisories(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected []AdvisoryJSON
	}{
		"empty string": {
			input:    "",
			expected: nil,
		},
		"empty array": {
			input:    "[]",
			expected: nil,
		},
		"single advisory": {
			input: `[{"id":"GHSA-xxx","severity":"CRITICAL","cvss":9.8,"source":"GitHub Advisory DB"}]`,
			expected: []AdvisoryJSON{
				{ID: "GHSA-xxx", Severity: "CRITICAL", CVSS: 9.8, Source: "GitHub Advisory DB"},
			},
		},
		"multiple advisories": {
			input: `[{"id":"GHSA-xxx","severity":"CRITICAL","cvss":9.8,"source":"GitHub Advisory DB"},{"id":"GO-xxx","severity":"UNKNOWN","cvss":0,"source":"Go Vulnerability DB"}]`,
			expected: []AdvisoryJSON{
				{ID: "GHSA-xxx", Severity: "CRITICAL", CVSS: 9.8, Source: "GitHub Advisory DB"},
				{ID: "GO-xxx", Severity: "UNKNOWN", CVSS: 0, Source: "Go Vulnerability DB"},
			},
		},
		"invalid JSON": {
			input:    `{invalid}`,
			expected: nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := ParseAdvisories(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPrimaryAdvisoryID(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected string
	}{
		"empty": {
			input:    "",
			expected: "",
		},
		"single advisory": {
			input:    `[{"id":"GHSA-xxx","severity":"CRITICAL","cvss":9.8,"source":"GitHub"}]`,
			expected: "GHSA-xxx",
		},
		"multiple advisories": {
			input:    `[{"id":"GHSA-xxx","severity":"CRITICAL","cvss":9.8,"source":"GitHub"},{"id":"GO-xxx","severity":"UNKNOWN","cvss":0,"source":"Go"}]`,
			expected: "GHSA-xxx",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := GetPrimaryAdvisoryID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetAllAdvisoryIDs(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected []string
	}{
		"empty": {
			input:    "",
			expected: nil,
		},
		"single advisory": {
			input:    `[{"id":"GHSA-xxx","severity":"CRITICAL","cvss":9.8,"source":"GitHub"}]`,
			expected: []string{"GHSA-xxx"},
		},
		"multiple advisories": {
			input:    `[{"id":"GHSA-xxx","severity":"CRITICAL","cvss":9.8,"source":"GitHub"},{"id":"GO-xxx","severity":"UNKNOWN","cvss":0,"source":"Go"}]`,
			expected: []string{"GHSA-xxx", "GO-xxx"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := GetAllAdvisoryIDs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAdvisoryJSONRoundTrip(t *testing.T) {
	advisories := []AdvisoryJSON{
		{ID: "GHSA-xxx", Severity: "CRITICAL", CVSS: 9.8, Source: "GitHub Advisory DB"},
		{ID: "GO-xxx", Severity: "UNKNOWN", CVSS: 0, Source: "Go Vulnerability DB"},
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(advisories)
	require.NoError(t, err)

	jsonStr := string(jsonBytes)

	// Parse back
	parsed := ParseAdvisories(jsonStr)
	assert.Equal(t, advisories, parsed)

	// Verify individual extractors
	assert.Equal(t, "GHSA-xxx", GetPrimaryAdvisoryID(jsonStr))
	assert.Equal(t, "GitHub Advisory DB", GetPrimarySourceName(jsonStr))
	assert.Equal(t, []string{"GHSA-xxx", "GO-xxx"}, GetAllAdvisoryIDs(jsonStr))
}
