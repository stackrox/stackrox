package tracker

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

type aggregationKey string // e.g. IMPORTANT_VULNERABILITY_SEVERITY|true

// aggregatedRecord counts the number of occurrences of a set of label values.
type aggregatedRecord struct {
	labels prometheus.Labels
	total  int
}

// aggregator is a Finding processor, that counts the number of occurences of
// every combination of label values in the findings.
// The processing result is stored in the result field.
// MetricDescriptors is a map of metric name to their sorted lists of labels.
//
// For example, for a metric M1 with labels L1 and L2, and metric M2 with a
// single label L2, provided the following findings:
//
//	[{L1="X", L2="Y"}, {L1="X", L2="Z"}, {L1="X", L2="Z"}],
//
// the aggregator will produce the following result:
//
//	{
//		M1:
//			{"X|Y": {labels: {L1="X", L2="Y"}, total: 1}},
//			{"X|Z": {labels: {L1="X", L2="Z"}, total: 2}}
//		M2:
//			{"Y": {labels: {L2="Y"}, total: 1}},
//			{"Z": {labels: {L2="Z"}, total: 2}}
//	}
type aggregator[F Finding] struct {
	result  map[MetricName]map[aggregationKey]*aggregatedRecord
	md      MetricDescriptors
	getters LazyLabelGetters[F]
}

func makeAggregator[F Finding](md MetricDescriptors, getters LazyLabelGetters[F]) *aggregator[F] {
	aggregated := make(map[MetricName]map[aggregationKey]*aggregatedRecord)
	for metric := range md {
		aggregated[metric] = make(map[aggregationKey]*aggregatedRecord)
	}
	return &aggregator[F]{aggregated, md, getters}
}

// count the finding in the aggregation result.
func (a *aggregator[F]) count(finding F) {
	increment := 1
	if f, ok := any(finding).(WithIncrement); ok {
		increment = f.GetIncrement()
	}

	for metric, labels := range a.md {
		key, labels := a.makeAggregationKey(labels, finding)
		if rec, ok := a.result[metric][key]; ok {
			rec.total += increment
		} else {
			a.result[metric][key] = &aggregatedRecord{labels, increment}
		}
	}
}

// makeAggregationKey computes an aggregation key according to the provided
// labels, and the map of the requested labels to their values.
// The values in the key are ordered according to the labels order.
//
// Example:
//
//	"Cluster,Deployment" => "pre-prod|backend", {"Cluster": "pre-prod", "Deployment": "backend")}
func (a *aggregator[F]) makeAggregationKey(labels []Label, finding F) (aggregationKey, prometheus.Labels) {
	vector := make(prometheus.Labels)
	var key strings.Builder
	for _, label := range labels {
		value := a.getters[label](finding)
		vector[string(label)] = value
		if key.Len() > 0 {
			key.WriteRune('|')
		}
		key.WriteString(value)
	}
	return aggregationKey(key.String()), vector
}
