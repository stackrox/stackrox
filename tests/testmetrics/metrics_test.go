//go:build test

package testmetrics

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_GetValue(t *testing.T) {
	testCases := map[string]struct {
		text   string
		name   string
		labels []string
		want   float64
		found  bool
	}{
		"should find simple metric": {
			text:  "some_counter_total 2\n",
			name:  "some_counter_total",
			want:  2,
			found: true,
		},
		"should find metric with labels": {
			text:   `labeled_total{status="ok"} 7` + "\n",
			name:   "labeled_total",
			labels: []string{"status", "ok"},
			want:   7,
			found:  true,
		},
		"should return not found for missing metric": {
			text:  "some_counter_total 2\n",
			name:  "absent_total",
			found: false,
		},
		"should return not found for wrong label value": {
			text:   `labeled_total{status="ok"} 7` + "\n",
			name:   "labeled_total",
			labels: []string{"status", "err"},
			found:  false,
		},
		"should parse value before optional timestamp": {
			text:  "some_counter_total 2 1718462400000\n",
			name:  "some_counter_total",
			want:  2,
			found: true,
		},
		"should return not found for empty input": {
			text:  "",
			name:  "foo_total",
			found: false,
		},
		"should not match label name that is a substring of another": {
			text:   `labeled_total{my_status="ok"} 7` + "\n",
			name:   "labeled_total",
			labels: []string{"status", "ok"},
			found:  false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			m := Parse(tc.text)
			val, found := m.GetValue(tc.name, tc.labels...)
			assert.Equal(t, tc.found, found)
			if tc.found {
				assert.Equal(t, tc.want, val)
			}
		})
	}
}

func TestParse_PrefixDoesNotFalseMatch(t *testing.T) {
	text := strings.Join([]string{
		"rox_scan_connections_total 5",
		`rox_scan_connections_errors_total{reason="timeout"} 3`,
	}, "\n")

	m := Parse(text)

	val, found := m.GetValue("rox_scan_connections_total")
	require.True(t, found)
	require.Equal(t, float64(5), val)

	val, found = m.GetValue("rox_scan_connections_errors_total", "reason", "timeout")
	require.True(t, found)
	require.Equal(t, float64(3), val)

	_, found = m.GetValue("rox_scan_connections")
	require.False(t, found, "rox_scan_connections should NOT match rox_scan_connections_total")
}
