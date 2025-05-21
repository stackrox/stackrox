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
	mle        MetricLabelsExpressions
	labelOrder map[Label]int
}

func makeAggregator(mle MetricLabelsExpressions, labelOrder map[Label]int) *aggregator {
	aggregated := make(map[MetricName]map[aggregationKey]*aggregatedRecord)
	for metric := range mle {
		aggregated[metric] = make(map[aggregationKey]*aggregatedRecord)
	}
	return &aggregator{aggregated, mle, labelOrder}
}

func (r *aggregator) count(getter func(Label) string) {
	for metric, expressions := range r.mle {
		if key, labels := makeAggregationKey(expressions, getter, r.labelOrder); key != "" {
			if rec, ok := r.result[metric][key]; ok {
				rec.total++
			} else {
				r.result[metric][key] = &aggregatedRecord{labels, 1}
			}
		}
	}
}
