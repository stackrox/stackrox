package telemetry

import (
	"slices"
	"strings"
)

// parseAggregationExpressions parses the aggregation configuraion, and builds a
// map, that associates every aggregation key to the list of vulnerability
// filtering expressions.
//
// Example:
//
//	"Namespace,Severity": map[metricName][]expression{
//	  "Namespace_Severity_total": {"Namespace", "Severity"},
//	}
func parseAggregationExpressions(expr string) map[metricName][]Label {
	result := make(map[metricName][]Label)

	for _, key := range strings.FieldsFunc(expr, func(r rune) bool { return r == '|' }) {
		var labels []Label
		for _, label := range strings.FieldsFunc(key, func(r rune) bool { return r == ',' }) {
			label = strings.Trim(label, " ")
			if label != "" {
				labels = append(labels, label)
			}
		}
		if len(labels) > 0 {
			metric := makeMetricName(labels)
			result[metric] = labels
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// makeAggregationKeyInstance computes an aggregation key according to the
// labels, and the map of the requested labels to their values.
//
// Example:
//
//	"Cluster,Deployment" => "pre-prod|backend", {"Cluster": "pre-prod", "Deployment": "backend")}
func makeAggregationKeyInstance(labels []Label, labelsGetter func(Label) string) (metricKey, map[Label]string) {
	sb := strings.Builder{}
	result := make(map[Label]string)
	for i, label := range labels {
		if i > 0 {
			sb.WriteRune('|')
		}
		value := labelsGetter(label)
		sb.WriteString(value)
		result[label] = value
	}
	return sb.String(), result
}

func makeMetricName(labels []Label) string {
	slices.SortFunc(labels, func(a, b Label) int {
		return labelOrder[a] - labelOrder[b]
	})
	return strings.Join(labels, "_") + "_total"
}
