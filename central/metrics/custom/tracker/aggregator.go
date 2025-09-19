package tracker

import (
	"iter"

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
// The labelOrder is used to compute the aggregationKey (i.e. pipe separated
// label values).
// MetricDescriptors provides the list of metrics with their sets of labels.
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
	result     map[MetricName]map[aggregationKey]*aggregatedRecord
	md         MetricDescriptors
	labelOrder map[Label]int
	getters    map[Label]func(F) string
}

func makeAggregator[F Finding](md MetricDescriptors, labelOrder map[Label]int, getters map[Label]func(F) string) *aggregator[F] {
	aggregated := make(map[MetricName]map[aggregationKey]*aggregatedRecord)
	for metric := range md {
		aggregated[metric] = make(map[aggregationKey]*aggregatedRecord)
	}
	return &aggregator[F]{aggregated, md, labelOrder, getters}
}

// count the finding in the aggregation result.
func (r *aggregator[Finding]) count(finding Finding) {
	labelValue := func(label Label) string {
		return r.getters[label](finding)
	}
	for metric, labels := range r.md {
		if key, labels := makeAggregationKey(labels, labelValue, r.labelOrder); key != "" {
			if rec, ok := r.result[metric][key]; ok {
				rec.total += finding.GetIncrement()
			} else {
				r.result[metric][key] = &aggregatedRecord{labels, finding.GetIncrement()}
			}
		}
	}
}

// makeAggregationKey computes an aggregation key according to the labels from
// the provided expression, and the map of the requested labels to their values.
// The values in the key are sorted according to the provided labelOrder map.
//
// Example:
//
//	"Cluster,Deployment" => "pre-prod|backend", {"Cluster": "pre-prod", "Deployment": "backend")}
func makeAggregationKey(labelExpression []Label, getter func(Label) string, labelOrder map[Label]int) (aggregationKey, prometheus.Labels) {
	labels := make(prometheus.Labels)
	values := make(orderedValues, len(labelExpression))
	for label, value := range collectMatchingLabels(labelExpression, getter) {
		labels[string(label)] = value
		values = append(values, valueOrder{labelOrder[label], value})
	}
	if len(labels) != len(labelExpression) {
		return "", nil
	}
	return aggregationKey(values.join('|')), labels
}

// collectMatchingLabels returns an iterator over the labels and the values.
func collectMatchingLabels(labels []Label, getter func(Label) string) iter.Seq2[Label, string] {
	return func(yield func(Label, string) bool) {
		for _, label := range labels {
			if !yield(label, getter(label)) {
				return
			}
		}
	}
}
