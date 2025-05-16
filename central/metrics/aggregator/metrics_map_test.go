package aggregator

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func Test_makeAggregationKeyInstance(t *testing.T) {
	testMetric := map[Label]string{
		"Cluster":   "value",
		"CVSS":      "7.4",
		"IsFixable": "false",
		"Namespace": "value",
	}
	labelsGetter := func(label Label) string {
		return testMetric[label]
	}
	t.Run("matching", func(t *testing.T) {
		key, labels := makeAggregationKeyInstance(
			map[Label][]*expression{
				"Cluster":   {{"=", "*al*"}},
				"CVSS":      {{">", "5"}},
				"IsFixable": {{"", ""}},
			},
			labelsGetter)
		assert.Equal(t, metricKey("value|7.4|false"), key)
		assert.Equal(t, prometheus.Labels{
			"Cluster":   "value",
			"CVSS":      "7.4",
			"IsFixable": "false",
		}, labels)
	})
	t.Run("not matching", func(t *testing.T) {
		key, labels := makeAggregationKeyInstance(
			map[Label][]*expression{
				"Cluster":   {{"=", "missing"}},
				"CVSS":      {{">", "5"}},
				"IsFixable": {{"", ""}},
			},
			labelsGetter,
		)
		assert.Equal(t, metricKey(""), key)
		assert.Nil(t, labels)
	})
	t.Run("matching second", func(t *testing.T) {
		key, labels := makeAggregationKeyInstance(
			map[Label][]*expression{
				"Cluster": {
					{"=", "nope"},
					{"=", "nape"},
					{op: "OR"},
					{"=", "*al*"},
					{op: "OR"},
					{"=", "*ol*"},
				},
				"CVSS":      {{">", "5"}},
				"IsFixable": {{"", ""}},
			},
			labelsGetter)
		assert.Equal(t, metricKey("value|7.4|false"), key)
		assert.Equal(t, prometheus.Labels{
			"Cluster":   "value",
			"CVSS":      "7.4",
			"IsFixable": "false",
		}, labels)
	})
	t.Run("no matching with OR", func(t *testing.T) {
		key, labels := makeAggregationKeyInstance(
			map[Label][]*expression{
				"Cluster": {
					{"=", "nope"},
					{"=", "nape"},
					{op: "OR"},
					{"=", "*ul*"},
					{op: "OR"},
					{"=", "*ol*"},
				},
				"CVSS":      {{">", "5"}},
				"IsFixable": nil,
			},
			labelsGetter)
		assert.Equal(t, metricKey(""), key)
		assert.Equal(t, prometheus.Labels(nil), labels)
	})
}

func Test_getMetricLabels(t *testing.T) {
	cases := []struct {
		expressions map[Label][]*expression
		labels      []string
	}{
		{
			map[Label][]*expression{},
			[]string(nil),
		},
		{
			map[Label][]*expression{
				"a": {{"=", "b"}}},
			[]string{"a"},
		},
		{
			map[Label][]*expression{
				"CVE":      {{"", ""}},
				"Severity": {{"=", "x"}},
				"Cluster":  {{">", "4"}},
			},
			[]string{"Cluster", "CVE", "Severity"},
		},
	}
	for _, c := range cases {
		assert.Equal(t, c.labels, getMetricLabels(c.expressions), c.expressions)
	}
}
