package common

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

var testLabelGetters = []LabelGetter[testDataIndex]{
	testDataGetter("test"),
	testDataGetter("Cluster"),
	testDataGetter("Namespace"),
	testDataGetter("CVE"),
	testDataGetter("Severity"),
	testDataGetter("CVSS"),
	testDataGetter("IsFixable"),
}

func testDataGetter(label Label) LabelGetter[testDataIndex] {
	return LabelGetter[testDataIndex]{
		label,
		func(i testDataIndex) string { return testData[i][label] }}
}

var testLabelOrder = makeLabelOrderMap(testLabelGetters)

func TestMakeAggregationKeyInstance(t *testing.T) {
	testMetric := map[Label]string{
		"Cluster":   "value",
		"CVSS":      "7.4",
		"IsFixable": "false",
		"Namespace": "value",
	}
	getter := func(label Label) string {
		return testMetric[label]
	}
	t.Run("matching", func(t *testing.T) {
		key, labels := makeAggregationKey(
			map[Label][]*Expression{
				"Cluster":   {{"=", "*al*"}},
				"CVSS":      {{">", "5"}},
				"IsFixable": {{"", ""}},
			},
			getter,
			testLabelOrder)
		assert.Equal(t, aggregationKey("value|7.4|false"), key)
		assert.Equal(t, prometheus.Labels{
			"Cluster":   "value",
			"CVSS":      "7.4",
			"IsFixable": "false",
		}, labels)
	})
	t.Run("not matching", func(t *testing.T) {
		key, labels := makeAggregationKey(
			map[Label][]*Expression{
				"Cluster":   {{"=", "missing"}},
				"CVSS":      {{">", "5"}},
				"IsFixable": {{"", ""}},
			},
			getter, testLabelOrder)
		assert.Equal(t, aggregationKey(""), key)
		assert.Nil(t, labels)
	})
	t.Run("matching second", func(t *testing.T) {
		key, labels := makeAggregationKey(
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
			getter, testLabelOrder)
		assert.Equal(t, aggregationKey("value|7.4|false"), key)
		assert.Equal(t, prometheus.Labels{
			"Cluster":   "value",
			"CVSS":      "7.4",
			"IsFixable": "false",
		}, labels)
	})
	t.Run("no matching with OR", func(t *testing.T) {
		key, labels := makeAggregationKey(
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
			getter, testLabelOrder)
		assert.Equal(t, aggregationKey(""), key)
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
