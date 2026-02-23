package resolvers

import (
	"testing"

	alertviews "github.com/stackrox/rox/central/alert/views"
	"github.com/stretchr/testify/assert"
)

func TestPolicySeverityCountsToResolver(t *testing.T) {
	cases := []struct {
		name     string
		counts   *alertviews.PolicySeverityCounts
		total    int32
		low      int32
		medium   int32
		high     int32
		critical int32
	}{
		{
			name:   "all zeros",
			counts: &alertviews.PolicySeverityCounts{},
		},
		{
			name: "mixed counts",
			counts: &alertviews.PolicySeverityCounts{
				LowCount:      2,
				MediumCount:   3,
				HighCount:     1,
				CriticalCount: 4,
			},
			total:    10,
			low:      2,
			medium:   3,
			high:     1,
			critical: 4,
		},
		{
			name: "single severity",
			counts: &alertviews.PolicySeverityCounts{
				CriticalCount: 7,
			},
			total:    7,
			critical: 7,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resolver := policySeverityCountsToResolver(tc.counts)
			assert.Equal(t, tc.total, resolver.total)
			assert.Equal(t, tc.low, resolver.low)
			assert.Equal(t, tc.medium, resolver.medium)
			assert.Equal(t, tc.high, resolver.high)
			assert.Equal(t, tc.critical, resolver.critical)
		})
	}
}
