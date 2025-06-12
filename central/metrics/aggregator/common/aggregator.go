package common

import (
	"github.com/prometheus/client_golang/prometheus"
)

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
