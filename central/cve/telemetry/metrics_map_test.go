package telemetry

import (
	"testing"

	"github.com/gobwas/glob"
	"github.com/stretchr/testify/assert"
)

func Test_makeMetricName(t *testing.T) {
	cases := map[aggregationKey]string{
		"":                       "_total",
		"Severity":               "Severity_total",
		"Cluster=*prod*,CVSS>=5": "Cluster_eq__prod__CVSS_gt__eq_5_total",
	}

	for key, name := range cases {
		assert.Equal(t, name, makeMetricName(key))
	}
}

func Test_parseAggregationKeys(t *testing.T) {
	keys := parseAggregationKeys("Namespace=abc,Severity,IsFixable=true|Cluster|SeverityV3")
	assert.Equal(t, map[aggregationKey][]string{
		"Cluster_total": {"Cluster"},
		"Namespace_eq_abc_Severity_IsFixable_eq_true_total": {"Namespace=abc", "Severity", "IsFixable=true"},
		"SeverityV3_total": {"SeverityV3"},
	}, keys)
}

func Test_makeAggregationKeyInstance(t *testing.T) {
	metric := map[string]string{
		"string": "value",
		"number": "7.4",
		"bool":   "false",
	}
	globCache = make(map[string]glob.Glob)
	assert.Equal(t, "value|7.4|false", makeAggregationKeyInstance(
		[]expression{"string=*al*", "number>5", "bool"}, metric))
}

func Test_getMetricNames(t *testing.T) {
	assert.Equal(t, []string(nil), getMetricLabels([]expression{}))
	assert.Equal(t, []string{"a"}, getMetricLabels([]expression{
		"a=b",
	}))
	assert.Equal(t, []string{"a", "b", "c"}, getMetricLabels([]expression{
		"a", "b=x", "c>4",
	}))
}
