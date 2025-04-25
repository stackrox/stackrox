package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_makeMetricName(t *testing.T) {
	cases := map[aggregationKey]metricName{
		"":                       "_total",
		"Severity":               "Severity_total",
		"Cluster=*prod*,CVSS>=5": "Cluster_eq__prod__CVSS_gt__eq_5_total",
	}

	for key, name := range cases {
		assert.Equal(t, name, makeMetricName(key))
	}
}

func Test_parseAggregationKeys(t *testing.T) {
	keys := parseAggregationExpressions("Namespace=abc, Severity, IsFixable=true | Cluster | SeverityV3")
	assert.Equal(t, map[metricName][]expression{
		"Cluster_total": {"Cluster"},
		"Namespace_eq_abc_Severity_IsFixable_eq_true_total": {"Namespace=abc", "Severity", "IsFixable=true"},
		"SeverityV3_total": {"SeverityV3"},
	}, keys)
}

func Test_makeAggregationKeyInstance(t *testing.T) {
	testMetric := map[string]string{
		"string":  "value",
		"number":  "7.4",
		"bool":    "false",
		"another": "value",
	}
	labelsGetter := func(label string) string {
		return testMetric[label]
	}
	key, labels := makeAggregationKeyInstance(
		[]expression{"string=*al*", "number>5", "bool"}, labelsGetter)
	assert.Equal(t, "value|7.4|false", key)
	assert.Equal(t, map[string]string{
		"string": "value",
		"number": "7.4",
		"bool":   "false",
	}, labels)
}

func Test_getMetricNames(t *testing.T) {
	cases := []struct {
		expressions []expression
		names       []string
	}{
		{
			[]expression{},
			[]string(nil),
		},
		{
			[]expression{"a=b"},
			[]string{"a"},
		},
		{
			[]expression{"a", "b=x", "c>4"},
			[]string{"a", "b", "c"},
		},
	}
	for _, c := range cases {
		assert.Equal(t, c.names, getMetricLabels(c.expressions), c.expressions)
	}
}
