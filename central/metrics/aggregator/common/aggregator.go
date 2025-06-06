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
type aggregator struct {
	result     map[MetricName]map[aggregationKey]*aggregatedRecord
	mcfg       MetricsConfiguration
	labelOrder map[Label]int
}

func makeAggregator(mcfg MetricsConfiguration, labelOrder map[Label]int) *aggregator {
	aggregated := make(map[MetricName]map[aggregationKey]*aggregatedRecord)
	for metric := range mcfg {
		aggregated[metric] = make(map[aggregationKey]*aggregatedRecord)
	}
	return &aggregator{aggregated, mcfg, labelOrder}
}

func (r *aggregator) count(getter func(Label) string, count int) {
	for metric, labels := range r.mcfg {
		if key, labels := makeAggregationKey(labels, getter, r.labelOrder); key != "" {
			if rec, ok := r.result[metric][key]; ok {
				rec.total += count
			} else {
				r.result[metric][key] = &aggregatedRecord{labels, count}
			}
		}
	}
}
