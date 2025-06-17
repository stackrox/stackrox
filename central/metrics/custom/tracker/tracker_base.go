package tracker

// LazyLabel enables deferred evaluation of a label's value.
// Computing and storing values for all labels for every finding would be
// inefficient. Instead, the Getter function computes the value for this
// specific label only when provided with a finding.
type LazyLabel[Finding any] struct {
	Label
	Getter func(*Finding) string
}

// MakeLabelOrderMap maps labels to their order according to the order of
// the labels in the list of getters.
// Respecting the order is important for computing the aggregation key, which is
// a concatenation of label values.
func MakeLabelOrderMap[Finding any](getters []LazyLabel[Finding]) map[Label]int {
	result := make(map[Label]int, len(getters))
	for i, getter := range getters {
		result[getter.Label] = i + 1
	}
	return result
}
