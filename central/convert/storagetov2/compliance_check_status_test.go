package storagetov2

import (
	"testing"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stretchr/testify/assert"
)

func TestDeriveAggregateDataFreshness(t *testing.T) {
	cases := map[string]struct {
		staleCount   int
		currentCount int
		expected     v2.ComplianceAggregateDataFreshness
	}{
		"no stale, no current -> UNKNOWN": {
			staleCount:   0,
			currentCount: 0,
			expected:     v2.ComplianceAggregateDataFreshness_COMPLIANCE_AGGREGATE_DATA_FRESHNESS_UNKNOWN,
		},
		"only current -> CURRENT": {
			staleCount:   0,
			currentCount: 5,
			expected:     v2.ComplianceAggregateDataFreshness_COMPLIANCE_AGGREGATE_DATA_FRESHNESS_CURRENT,
		},
		"only stale -> STALE": {
			staleCount:   3,
			currentCount: 0,
			expected:     v2.ComplianceAggregateDataFreshness_COMPLIANCE_AGGREGATE_DATA_FRESHNESS_STALE,
		},
		"both stale and current -> INCONSISTENT": {
			staleCount:   3,
			currentCount: 5,
			expected:     v2.ComplianceAggregateDataFreshness_COMPLIANCE_AGGREGATE_DATA_FRESHNESS_INCONSISTENT,
		},
		// Regression test for the bug this replaces: deriving from MIN/MAX of the
		// stored freshness enum reported STALE (not INCONSISTENT) whenever UNKNOWN
		// rows were also present, because MIN dropped to UNKNOWN. Counts sidestep
		// this entirely: UNKNOWN rows are not counted by either staleCount or
		// currentCount, so any number of them does not change the outcome below.
		"stale and current mixed with unknown rows present -> INCONSISTENT": {
			staleCount:   2,
			currentCount: 2,
			expected:     v2.ComplianceAggregateDataFreshness_COMPLIANCE_AGGREGATE_DATA_FRESHNESS_INCONSISTENT,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, DeriveAggregateDataFreshness(tc.staleCount, tc.currentCount))
		})
	}
}
