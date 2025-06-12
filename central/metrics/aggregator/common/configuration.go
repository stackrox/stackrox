package common

import (
	"slices"
)

type Label string      // Prometheus label.
type MetricName string // Prometheus metric name.

type LabelGetter[Finding Countable] struct {
	Label  Label
	Getter func(Finding) string
}

// MetricsConfiguration is the parsed aggregation configuration.
type MetricsConfiguration map[MetricName]map[Label]Expression

func (mcfg MetricsConfiguration) HasAnyLabelOf(labels []Label) bool {
	for _, labelExpr := range mcfg {
		for label := range labelExpr {
			if slices.Contains(labels, label) {
				return true
			}
		}
	}
	return false
}

// MakeLabelOrderMap maps labels to their order according to the order of
// the labels in the list of getters.
func MakeLabelOrderMap[Finding Countable](getters []LabelGetter[Finding]) map[Label]int {
	result := make(map[Label]int, len(getters))
	for i, getter := range getters {
		result[getter.Label] = i + 1
	}
	return result
}
