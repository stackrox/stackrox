package tracker

// LazyLabel allows for lazy label value evaluation.
// A metric labels is usually a subset of all available labels for this metric
// category. That would be inefficient to compute and store in memory values of
// all labels per finding.
// The Getter function, provided with a finding, returns the value only for the
// wrapped label.
type LazyLabel[Finding any] struct {
	Label
	Getter func(*Finding) string
}

// MakeLabelOrderMap maps labels to their order according to the order of
// the labels in the list of getters.
func MakeLabelOrderMap[Finding any](getters []LazyLabel[Finding]) map[Label]int {
	result := make(map[Label]int, len(getters))
	for i, getter := range getters {
		result[getter.Label] = i + 1
	}
	return result
}
