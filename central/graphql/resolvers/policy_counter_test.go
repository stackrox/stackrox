package resolvers

import (
	"testing"

	alertviews "github.com/stackrox/rox/central/alert/views"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestMapListAlertsToPolicyCounterResolver(t *testing.T) {
	alerts := []*storage.ListAlert{
		{
			State: storage.ViolationState_ACTIVE,
			Policy: &storage.ListAlertPolicy{
				Id:       "id1",
				Severity: storage.Severity_LOW_SEVERITY,
			},
		},
		{
			State: storage.ViolationState_ACTIVE,
			Policy: &storage.ListAlertPolicy{
				Id:       "id2",
				Severity: storage.Severity_HIGH_SEVERITY,
			},
		},
		{
			State: storage.ViolationState_RESOLVED,
			Policy: &storage.ListAlertPolicy{
				Id:       "id3",
				Severity: storage.Severity_CRITICAL_SEVERITY,
			},
		},
		{
			State: storage.ViolationState_ACTIVE,
			Policy: &storage.ListAlertPolicy{
				Id:       "id1",
				Severity: storage.Severity_LOW_SEVERITY,
			},
		},
		{
			State: storage.ViolationState_ACTIVE,
			Policy: &storage.ListAlertPolicy{
				Id:       "id3",
				Severity: storage.Severity_LOW_SEVERITY,
			},
		},
	}

	counterResolver := mapListAlertsToPolicySeverityCount(alerts)
	assert.Equal(t, int32(3), counterResolver.total)
	assert.Equal(t, int32(2), counterResolver.low)
	assert.Equal(t, int32(0), counterResolver.medium)
	assert.Equal(t, int32(1), counterResolver.high)
	assert.Equal(t, int32(0), counterResolver.critical)
}

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
