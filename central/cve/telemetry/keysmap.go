package telemetry

import (
	"strings"
)

type expression = string

var keysMap map[aggregationKey][]expression

func parseAggregationKeys(setting string) map[aggregationKey][]expression {
	result := make(map[aggregationKey][]expression)
	for _, key := range strings.Split(setting, "|") {
		result[key] = strings.Split(key, ",")
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
