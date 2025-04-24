package telemetry

import (
	"strings"
)

type expression = string

var metricsMap map[aggregationKey][]expression

var opNames = map[rune]string{
	'=': "_eq_",
	'!': "_not_",
	'>': "_gt_",
	'<': "_lt_",
}

func makeMetricName(key aggregationKey) string {
	result := strings.Builder{}
	for _, u := range key {
		if u >= 'a' && u <= 'z' || u >= 'A' && u <= 'Z' || u >= '0' && u <= '9' {
			result.WriteRune(u)
		} else {
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

func parseAggregationKeys(setting string) map[aggregationKey][]expression {
	result := make(map[aggregationKey][]expression)
	for _, key := range strings.Split(setting, "|") {
		result[makeMetricName(key)] = strings.Split(key, ",")
	}
	return result
}

func makeAggregationKeyInstance(expressions []expression, metric map[keyInstance]string) string {
	sb := strings.Builder{}
	for i, expr := range expressions {
		key, ok := filter(expr, metric)
		if !ok {
			return ""
		}
		if v := metric[key]; v != "" {
			if i > 0 {
				sb.WriteRune('|')
			}
			sb.WriteString(v)
		} else {
			return ""
		}
	}
	return sb.String()
}

func getMetricLabels(expressions []expression) []string {
	var labels []string
	for _, expression := range expressions {
		label, _, _ := splitExpression(expression)
		labels = append(labels, label)
	}
	return labels
}
