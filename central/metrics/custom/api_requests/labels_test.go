package api_requests

import (
	"net/http"
	"testing"

	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/pkg/glob"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_compareIgnoringDash(t *testing.T) {
	cases := map[string]struct {
		header string
		label  tracker.Label
		equal  bool
	}{
		"exact match":           {"Accept", "Accept", true},
		"single dash":           {"User-Agent", "UserAgent", true},
		"multiple dashes":       {"Rh-Servicenow-Instance", "RhServicenowInstance", true},
		"leading dash":          {"-Leading", "Leading", true},
		"trailing dash":         {"Trailing-", "Trailing", true},
		"consecutive dashes":    {"A--B", "AB", true},
		"all dashes":            {"---", "", true},
		"both empty":            {"", "", true},
		"header shorter":        {"Rh", "RhServicenow", false},
		"label shorter":         {"Rh-Servicenow-Instance", "Rh", false},
		"different characters":  {"X-Custom", "YCustom", false},
		"dash only vs nonempty": {"-", "A", false},
		"nonempty vs empty":     {"A", "", false},
		"empty vs nonempty":     {"", "A", false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.equal, compareIgnoringDash(tc.header, tc.label))
		})
	}
}

func Test_makeHeaderGetter(t *testing.T) {
	cases := map[string]struct {
		headerPattern glob.Pattern
		valuePattern  glob.Pattern
		label         tracker.Label
		finding       *finding
		expected      string
	}{
		"exact header, any value": {
			headerPattern: "Rh-Servicenow-Instance",
			valuePattern:  "",
			label:         "RhServicenowInstance",
			finding:       &finding{Headers: http.Header{"Rh-Servicenow-Instance": {"prod.service-now.com"}}},
			expected:      "prod.service-now.com",
		},
		"exact header, value filter matches": {
			headerPattern: "User-Agent",
			valuePattern:  "*ServiceNow*",
			label:         "UserAgent",
			finding:       &finding{Headers: http.Header{"User-Agent": {"Mozilla ServiceNow Bot"}}},
			expected:      "Mozilla ServiceNow Bot",
		},
		"exact header, value filter rejects": {
			headerPattern: "User-Agent",
			valuePattern:  "*ServiceNow*",
			label:         "UserAgent",
			finding:       &finding{Headers: http.Header{"User-Agent": {"curl/8.0"}}},
			expected:      "",
		},
		"exact header, multiple values partially filtered": {
			headerPattern: "User-Agent",
			valuePattern:  "roxctl/*",
			label:         "UserAgent",
			finding:       &finding{Headers: http.Header{"User-Agent": {"roxctl/4.5", "gateway"}}},
			expected:      "roxctl/4.5",
		},
		"glob header pattern": {
			headerPattern: "Rh-*",
			valuePattern:  "*",
			label:         "RhServicenowInstance",
			finding:       &finding{Headers: http.Header{"Rh-Servicenow-Instance": {"prod"}}},
			expected:      "prod",
		},
		"glob header, label selects correct header": {
			headerPattern: "Rh-*",
			valuePattern:  "*",
			label:         "RhRoxctlCommand",
			finding: &finding{Headers: http.Header{
				"Rh-Servicenow-Instance": {"prod"},
				"Rh-Roxctl-Command":      {"check image"},
			}},
			expected: "check image",
		},
		"header absent": {
			headerPattern: "Rh-Servicenow-Instance",
			valuePattern:  "",
			label:         "RhServicenowInstance",
			finding:       &finding{Headers: http.Header{"User-Agent": {"curl/8.0"}}},
			expected:      "",
		},
		"no headers at all": {
			headerPattern: "Rh-Servicenow-Instance",
			valuePattern:  "",
			label:         "RhServicenowInstance",
			finding:       &finding{Headers: http.Header{}},
			expected:      "",
		},
		"multiple values joined": {
			headerPattern: "X-Custom",
			valuePattern:  "",
			label:         "XCustom",
			finding:       &finding{Headers: http.Header{"X-Custom": {"a", "b", "c"}}},
			expected:      "a; b; c",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, tc.headerPattern.Compile())
			require.NoError(t, tc.valuePattern.Compile())
			getter := makeHeaderGetter(tc.headerPattern, tc.valuePattern, tc.label)
			assert.Equal(t, tc.expected, getter(tc.finding))
		})
	}
}
