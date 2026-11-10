package storagetov2

import (
	"testing"

	"github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestConvertComplianceDataFreshness(t *testing.T) {
	cases := map[string]struct {
		input    storage.ComplianceOperatorCheckResultV2_DataFreshness
		expected v2.ComplianceCheckDataFreshness
	}{
		"UNKNOWN": {
			input:    storage.ComplianceOperatorCheckResultV2_UNKNOWN,
			expected: v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_UNKNOWN,
		},
		"CURRENT": {
			input:    storage.ComplianceOperatorCheckResultV2_CURRENT,
			expected: v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_CURRENT,
		},
		"STALE": {
			input:    storage.ComplianceOperatorCheckResultV2_STALE,
			expected: v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_STALE,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, convertComplianceDataFreshness(tc.input))
		})
	}
}

func TestAggregateDataFreshness(t *testing.T) {
	cases := map[string]struct {
		currentCount int
		staleCount   int
		expected     v2.ComplianceAggregateDataFreshness
	}{
		"none reported is unknown": {
			expected: v2.ComplianceAggregateDataFreshness_COMPLIANCE_AGGREGATE_DATA_FRESHNESS_UNKNOWN,
		},
		"only current is current": {
			currentCount: 3,
			expected:     v2.ComplianceAggregateDataFreshness_COMPLIANCE_AGGREGATE_DATA_FRESHNESS_CURRENT,
		},
		"only stale is stale": {
			staleCount: 2,
			expected:   v2.ComplianceAggregateDataFreshness_COMPLIANCE_AGGREGATE_DATA_FRESHNESS_STALE,
		},
		"mixed is inconsistent": {
			currentCount: 3,
			staleCount:   2,
			expected:     v2.ComplianceAggregateDataFreshness_COMPLIANCE_AGGREGATE_DATA_FRESHNESS_INCONSISTENT,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, aggregateDataFreshness(tc.currentCount, tc.staleCount))
		})
	}
}

func TestCheckResultCarriesDataFreshness(t *testing.T) {
	result := &storage.ComplianceOperatorCheckResultV2{
		CheckId:       "check-1",
		CheckName:     "check-name",
		Status:        storage.ComplianceOperatorCheckResultV2_PASS,
		DataFreshness: storage.ComplianceOperatorCheckResultV2_STALE,
	}

	assert.Equal(t, v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_STALE,
		checkResult(result, "rule", nil).GetDataFreshness())
	assert.Equal(t, v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_STALE,
		clusterStatus(result, nil).GetDataFreshness())
}

func TestProfileStatsCarriesStaleCounts(t *testing.T) {
	counts := []*datastore.ResourceResultCountByProfile{
		{
			ProfileName:       "profile-1",
			StaleCount:        7,
			StaleClusterCount: 2,
		},
	}
	profileMap := map[string]*storage.ComplianceOperatorProfileV2{
		"profile-1": {Name: "profile-1"},
	}

	stats := ComplianceV2ProfileStats(counts, profileMap, nil)
	assert.Len(t, stats, 1)
	assert.Equal(t, int32(2), stats[0].GetStaleClusterCount())
	assert.Equal(t, int32(7), stats[0].GetStaleCheckResultCount())
}
