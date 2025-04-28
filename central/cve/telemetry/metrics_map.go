package telemetry

import (
	"strings"
)

type aggregationKey string // e.g. "Severity|IsFixable"

var opNames = map[rune]string{
	'=': "_eq_",
	'!': "_not_",
	'>': "_gt_",
	'<': "_lt_",
}

// makeMetricName sanitizes the metric key according to the Prometheus naming
// rules, which is [a-zA-Z0-9_]+ (no colon).
//
// Example:
//
//	"CVSS>5,Cluster=*prod": "CVSS_gt_5_Cluster_eq__prod_total"
func makeMetricName(key aggregationKey) metricName {
	result := strings.Builder{}
	for _, u := range key {
		if u >= 'a' && u <= 'z' || u >= 'A' && u <= 'Z' || u >= '0' && u <= '9' {
			result.WriteRune(u)
		} else if u != ' ' {
			if op, ok := opNames[u]; ok {
				result.WriteString(op)
			} else {
				result.WriteRune('_')
			}
		}
	}
	result.WriteString("_total")
	return result.String()
}

// parseAggregationExpressions parses the aggregation configuraion, and builds a
// map, that associates every aggregation key to the list of vulnerability
// filtering expressions.
//
// Example:
//
//	"Namespace=abc,Severity": map[metricName][]expression{
//	  "Namespace_eq_abc_Severity_total": {"Namespace=abc", "Severity"},
//	}
func parseAggregationExpressions(keys string) map[metricName][]*expression {
	result := make(map[metricName][]*expression)

	for _, key := range strings.FieldsFunc(keys, func(r rune) bool { return r == '|' }) {
		var expressions []*expression
		var keys []string
		for _, expr := range strings.FieldsFunc(key, func(r rune) bool { return r == ',' }) {
			expr := makeExpression(expr)
			if expr.label != "" {
				expressions = append(expressions, expr)
				// reconstruct the sanitized expression:
				keys = append(keys, expr.String())
			}
		}
		if len(expressions) > 0 {
			result[makeMetricName(aggregationKey(strings.Join(keys, "|")))] = expressions
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// makeAggregationKeyInstance computes an aggregation key according to the
// labels from the provided expressions, and the map of the requested labels
// to their values.
//
// Example:
//
//	"Cluster=*prod,Deployment" => "pre-prod|backend", {"Cluster": "pre-prod", "Deployment": "backend")}
func makeAggregationKeyInstance(expressions []*expression, labelsGetter func(string) string) (metricKey, map[string]string) {
	sb := strings.Builder{}
	labels := make(map[string]string)
	for i, expr := range expressions {
		value, ok := expr.match(labelsGetter)
		if !ok {
			return "", nil
		}
		if v := value; v != "" {
			if i > 0 {
				sb.WriteRune('|')
			}
			sb.WriteString(v)
			labels[expr.label] = value
		} else {
			return "", nil
		}
	}
	return sb.String(), labels
}

// getMetricLabels extracts the metric labels from the filter expressions.
//
// Example:
//
//	"Cluster=*prod,Namespace" => {"Cluster", "Namespace"}
func getMetricLabels(expressions []*expression) []string {
	var labels []string
	for _, expression := range expressions {
		labels = append(labels, expression.label)
	}
	return labels
}
