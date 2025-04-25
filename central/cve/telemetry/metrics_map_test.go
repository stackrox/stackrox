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

func str2expr(expr ...string) []expression {
	result := make([]expression, 0, len(expr))
	for _, e := range expr {
		result = append(result, makeExpression(e))
	}
	return result
}

func Test_parseAggregationExpressions(t *testing.T) {
	cases := map[string]map[metricName][]expression{
		// Default case:
		"Cluster,Namespace,Severity": {
			"Cluster_Namespace_Severity_total": str2expr("Cluster", "Namespace", "Severity"),
		},
		// Normal case:
		"Namespace=abc, Severity, IsFixable=true | Cluster | SeverityV3": {
			"Cluster_total": str2expr("Cluster"),
			"Namespace_eq_abc_Severity_IsFixable_eq_true_total": str2expr(
				"Namespace=abc",
				"Severity",
				"IsFixable=true"),
			"SeverityV3_total": str2expr("SeverityV3"),
		},

		// Weird cases:
		"":  nil,
		",": nil,
		"key,": {
			"key_total": str2expr("key"),
		},
		", key1 = x ,,||, key2  > 3|": {
			"key1_eq_x_total": str2expr("key1=x"),
			"key2_gt_3_total": str2expr("key2>3"),
		},
	}
	for input, expressions := range cases {
		assert.Equal(t, expressions, parseAggregationExpressions(input))
	}
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
		str2expr("string=*al*", "number>5", "bool"), labelsGetter)
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
			str2expr("a=b"),
			[]string{"a"},
		},
		{
			str2expr("a", "b=x", "c>4"),
			[]string{"a", "b", "c"},
		},
	}
	for _, c := range cases {
		assert.Equal(t, c.names, getMetricLabels(c.expressions), c.expressions)
	}
}
