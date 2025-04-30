package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_makeMetricName(t *testing.T) {
	assert.Equal(t, "_total",
		makeMetricName([]Label{""}))
	assert.Equal(t, "Namespace_total",
		makeMetricName([]Label{"Namespace"}))
	assert.Equal(t, "Cluster_Namespace_Severity_total",
		makeMetricName([]Label{"Severity", "Namespace", "Cluster"}))
	assert.Equal(t, "bad label_total",
		makeMetricName([]Label{"bad label"}))
}

func Test_parseAggregationExpressions(t *testing.T) {
	cases := map[string]map[metricName][]Label{
		// Default case:
		"Cluster,Namespace,Severity": {
			"Cluster_Namespace_Severity_total": {"Cluster", "Namespace", "Severity"},
		},
		// Weird cases:
		"":  nil,
		",": nil,
		"key,": {
			"key_total": {"key"},
		},
		", key1  ,,||, key2  |": {
			"key1_total": {"key1"},
			"key2_total": {"key2"},
		},
	}
	for input, expressions := range cases {
		assert.Equal(t, expressions, parseAggregationExpressions(input))
	}
}

func Test_makeAggregationKeyInstance(t *testing.T) {
	testMetric := map[Label]string{
		"string":  "value",
		"number":  "7.4",
		"bool":    "false",
		"another": "value",
	}
	labelsGetter := func(label Label) string {
		return testMetric[label]
	}
	t.Run("matching", func(t *testing.T) {
		key, labels := makeAggregationKeyInstance(
			[]Label{"string", "number", "bool"}, labelsGetter)
		assert.Equal(t, "value|7.4|false", key)
		assert.Equal(t, map[string]string{
			"string": "value",
			"number": "7.4",
			"bool":   "false",
		}, labels)
	})
	t.Run("missing", func(t *testing.T) {
		key, labels := makeAggregationKeyInstance(
			[]Label{"one", "two", "bool"},
			labelsGetter,
		)
		assert.Equal(t, "||false", key)
		assert.Equal(t, map[string]string{
			"one":  "",
			"two":  "",
			"bool": "false",
		}, labels)
	})
}
