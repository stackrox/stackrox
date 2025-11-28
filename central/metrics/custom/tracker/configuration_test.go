package tracker

import (
	"regexp"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func Test_allMetricsFilterActiveState(t *testing.T) {
	activeStateStr := storage.ViolationState_ACTIVE.String()
	resolvedStateStr := storage.ViolationState_RESOLVED.String()

	tests := []struct {
		name     string
		metrics  MetricDescriptors
		filters  LabelFilters
		expected bool
	}{
		{
			name:     "nil metrics and filters",
			metrics:  nil,
			filters:  nil,
			expected: false,
		},
		{
			name:     "empty metrics",
			metrics:  MetricDescriptors{},
			filters:  LabelFilters{},
			expected: false,
		},
		{
			name: "single metric and nil filters",
			metrics: MetricDescriptors{
				"metric1": {"Cluster", "State"},
			},
			filters:  nil,
			expected: false,
		},
		{
			name: "single metric with ^ACTIVE$ filter",
			metrics: MetricDescriptors{
				"metric1": {"Cluster", "State"},
			},
			filters: LabelFilters{
				"metric1": {
					Label("State"): regexp.MustCompile("^" + activeStateStr + "$"),
				},
			},
			expected: true,
		},
		{
			name: "single metric without State filter",
			metrics: MetricDescriptors{
				"metric1": {"Cluster", "Severity"},
			},
			filters: LabelFilters{
				"metric1": {
					Label("Cluster"): regexp.MustCompile("prod"),
				},
			},
			expected: false,
		},
		{
			name: "single metric with State filter that doesn't match ACTIVE",
			metrics: MetricDescriptors{
				"metric1": {"State"},
			},
			filters: LabelFilters{
				"metric1": {
					Label("State"): regexp.MustCompile("^" + resolvedStateStr + "$"),
				},
			},
			expected: false,
		},
		{
			name: "multiple metrics all with ACTIVE filter",
			metrics: MetricDescriptors{
				"metric1": {"Cluster", "State"},
				"metric2": {"Namespace", "State"},
			},
			filters: LabelFilters{
				"metric1": {
					Label("State"): regexp.MustCompile("^" + activeStateStr + "$"),
				},
				"metric2": {
					Label("State"): regexp.MustCompile("^" + activeStateStr + "$"),
				},
			},
			expected: true,
		},
		{
			name: "multiple metrics - only one has ACTIVE filter",
			metrics: MetricDescriptors{
				"metric1": {"Cluster", "State"},
				"metric2": {"Namespace"},
			},
			filters: LabelFilters{
				"metric1": {
					Label("State"): regexp.MustCompile("^" + activeStateStr + "$"),
				},
			},
			expected: false,
		},
		{
			name: "multiple metrics with mixed State filters",
			metrics: MetricDescriptors{
				"metric1": {"State"},
				"metric2": {"State"},
			},
			filters: LabelFilters{
				"metric1": {
					Label("State"): regexp.MustCompile("^" + activeStateStr + "$"),
				},
				"metric2": {
					Label("State"): regexp.MustCompile("^" + resolvedStateStr + "$"),
				},
			},
			expected: false,
		},
		{
			name: "single metric with unanchored pattern (should fail)",
			metrics: MetricDescriptors{
				"metric1": {"State"},
			},
			filters: LabelFilters{
				"metric1": {
					// String() returns "ACTIVE", but function expects "^ACTIVE$"
					Label("State"): regexp.MustCompile(activeStateStr),
				},
			},
			expected: false,
		},
		{
			name: "metric with ACTIVE|ATTEMPTED filter (should fail - different pattern)",
			metrics: MetricDescriptors{
				"metric1": {"State"},
			},
			filters: LabelFilters{
				"metric1": {
					// String() returns "^(ACTIVE|ATTEMPTED)$", not "^ACTIVE$"
					Label("State"): regexp.MustCompile("^(ACTIVE|ATTEMPTED)$"),
				},
			},
			expected: false,
		},
		{
			name: "metric with no filter for State label",
			metrics: MetricDescriptors{
				"metric1": {"State", "Cluster"},
			},
			filters: LabelFilters{
				"metric1": {
					Label("Cluster"): regexp.MustCompile("prod"),
					// No State filter
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Configuration{metrics: tt.metrics, filters: tt.filters}
			result := cfg.AllMetricsHaveFilter(Label("State"), "^ACTIVE$")
			assert.Equal(t, tt.expected, result)
		})
	}
}
