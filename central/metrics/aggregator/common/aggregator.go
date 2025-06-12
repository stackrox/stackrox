package common

import (
	"iter"

	"github.com/prometheus/client_golang/prometheus"
)

type aggregationKey string // e.g. IMPORTANT_VULNERABILITY_SEVERITY|true

// aggregatedRecord is a single gauge metric record.
type aggregatedRecord struct {
	labels prometheus.Labels
	total  int
}

// aggregator computes the aggregation result.
type aggregator[Finding Countable] struct {
	result     map[MetricName]map[aggregationKey]*aggregatedRecord
	mcfg       MetricsConfiguration
	labelOrder map[Label]int
	getters    map[Label]func(Finding) string
}

func makeAggregator[Finding Countable](mcfg MetricsConfiguration, labelOrder map[Label]int, getters map[Label]func(Finding) string) *aggregator[Finding] {
	aggregated := make(map[MetricName]map[aggregationKey]*aggregatedRecord)
	for metric := range mcfg {
		aggregated[metric] = make(map[aggregationKey]*aggregatedRecord)
	}
	return &aggregator[Finding]{aggregated, mcfg, labelOrder, getters}
}

// count the finding in the aggregation result.
func (r *aggregator[Finding]) count(finding Finding) {
	labelValue := func(label Label) string {
		return r.getters[label](finding)
	}
	for metric, labels := range r.mcfg {
		if key, labels := makeAggregationKey(labels, labelValue, r.labelOrder); key != "" {
			if rec, ok := r.result[metric][key]; ok {
				rec.total += finding.Count()
			} else {
				r.result[metric][key] = &aggregatedRecord{labels, finding.Count()}
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
//	"Cluster=*prod,Deployment" => "pre-prod|backend", {"Cluster": "pre-prod", "Deployment": "backend")}
func makeAggregationKey(labelExpression map[Label]Expression, getter func(Label) string, labelOrder map[Label]int) (aggregationKey, prometheus.Labels) {
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

// collectMatchingLabels returns an iterator over the labels and the values that
// match the expressions.
func collectMatchingLabels(labelExpression map[Label]Expression, getter func(Label) string) iter.Seq2[Label, string] {
	return func(yield func(Label, string) bool) {
		for label, expression := range labelExpression {
			value := getter(label)
			if expression.match(value) && !yield(label, value) {
				return
			}
		}
	}
}
