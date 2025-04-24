package telemetry

import (
	"testing"

	"github.com/gobwas/glob"
	"github.com/stretchr/testify/assert"
)

func Test_parseAggregationKeys(t *testing.T) {
	keys := parseAggregationKeys("Namespace=abc,Severity,IsFixable=true|Cluster|SeverityV3")
	assert.Equal(t, map[aggregationKey][]string{
		"Cluster":                               {"Cluster"},
		"Namespace=abc,Severity,IsFixable=true": {"Namespace=abc", "Severity", "IsFixable=true"},
		"SeverityV3":                            {"SeverityV3"},
	}, keys)
}

func Test_makeAggregationKeyInstance(t *testing.T) {
	metric := map[string]string{
		"string": "value",
		"number": "7.4",
		"bool":   "false",
	}
	globCache = make(map[string]glob.Glob)
	assert.Equal(t, "string=value|number=7.4|bool=false", makeAggregationKeyInstance([]expression{"string=*al*", "number>5", "bool"}, metric))
}
