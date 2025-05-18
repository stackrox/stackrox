package common

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

var testLabelOrder = map[Label]int{
	"test":      1,
	"Cluster":   2,
	"Namespace": 3,
	"CVE":       4,
	"Severity":  5,
	"CVSS":      6,
	"IsFixable": 7,
}

func TestMakeAggregationKeyInstance(t *testing.T) {
	testMetric := map[Label]string{
		"Cluster":   "value",
		"CVSS":      "7.4",
		"IsFixable": "false",
		"Namespace": "value",
	}
	finding := func(label Label) string {
		return testMetric[label]
	}
	t.Run("matching", func(t *testing.T) {
		key, labels := makeAggregationKeyInstance(
			map[Label][]*Expression{
				"Cluster":   {{"=", "*al*"}},
				"CVSS":      {{">", "5"}},
				"IsFixable": {{"", ""}},
			},
			finding, testLabelOrder)
		assert.Equal(t, metricKey("value|7.4|false"), key)
		assert.Equal(t, prometheus.Labels{
			"Cluster":   "value",
			"CVSS":      "7.4",
			"IsFixable": "false",
		}, labels)
	})
	t.Run("not matching", func(t *testing.T) {
		key, labels := makeAggregationKeyInstance(
			map[Label][]*Expression{
				"Cluster":   {{"=", "missing"}},
				"CVSS":      {{">", "5"}},
				"IsFixable": {{"", ""}},
			},
			finding, testLabelOrder)
		assert.Equal(t, metricKey(""), key)
		assert.Nil(t, labels)
	})
	t.Run("matching second", func(t *testing.T) {
		key, labels := makeAggregationKeyInstance(
			map[Label][]*Expression{
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
			finding, testLabelOrder)
		assert.Equal(t, metricKey("value|7.4|false"), key)
		assert.Equal(t, prometheus.Labels{
			"Cluster":   "value",
			"CVSS":      "7.4",
			"IsFixable": "false",
		}, labels)
	})
	t.Run("no matching with OR", func(t *testing.T) {
		key, labels := makeAggregationKeyInstance(
			map[Label][]*Expression{
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
			finding, testLabelOrder)
		assert.Equal(t, metricKey(""), key)
		assert.Equal(t, prometheus.Labels(nil), labels)
	})
}

func Test_getMetricLabels(t *testing.T) {
	cases := []struct {
		expressions map[Label][]*Expression
		labels      []string
	}{
		{
			map[Label][]*Expression{},
			[]string(nil),
		},
		{
			map[Label][]*Expression{
				"a": {{"=", "b"}}},
			[]string{"a"},
		},
		{
			map[Label][]*Expression{
				"CVE":      {{"", ""}},
				"Severity": {{"=", "x"}},
				"Cluster":  {{">", "4"}},
			},
			[]string{"Cluster", "CVE", "Severity"},
		},
	}
	for _, c := range cases {
		assert.Equal(t, c.labels, getMetricLabels(c.expressions, testLabelOrder), c.expressions)
	}
}
