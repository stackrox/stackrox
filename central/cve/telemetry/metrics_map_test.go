package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			[]*expression{
				{"string", "=", "*al*"},
				{"number", ">", "5"},
				{"bool", "", ""},
			},
			labelsGetter)
		assert.Equal(t, metricKey("value|7.4|false"), key)
		assert.Equal(t, map[string]string{
			"string": "value",
			"number": "7.4",
			"bool":   "false",
		}, labels)
	})
	t.Run("not matching", func(t *testing.T) {
		key, labels := makeAggregationKeyInstance(
			[]*expression{
				{"string", "=", "missing"},
				{"number", ">", "5"},
				{"bool", "", ""},
			},
			labelsGetter,
		)
		assert.Equal(t, metricKey(""), key)
		assert.Nil(t, labels)
	})
	t.Run("matching second", func(t *testing.T) {
		key, labels := makeAggregationKeyInstance(
			[]*expression{
				{"string", "=", "nope"},
				{"number", ">", "5"},
				nil,
				{"string", "=", "*al*"},
				nil,
				{"bool", "", ""},
			},
			labelsGetter)
		assert.Equal(t, metricKey("value"), key)
		assert.Equal(t, map[string]string{
			"string": "value",
		}, labels)
	})
}

func Test_getMetricLabels(t *testing.T) {
	cases := []struct {
		expressions []*expression
		labels      []string
	}{
		{
			[]*expression{},
			[]string(nil),
		},
		{
			[]*expression{{"a", "=", "b"}},
			[]string{"a"},
		},
		{
			[]*expression{
				{"a", "", ""},
				{"b", "=", "x"},
				{"c", ">", "4"},
			},
			[]string{"a", "b", "c"},
		},
	}
	for _, c := range cases {
		assert.Equal(t, c.labels, getMetricLabels(c.expressions), c.expressions)
	}
}
