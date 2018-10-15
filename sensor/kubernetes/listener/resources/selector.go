package resources

import "k8s.io/apimachinery/pkg/labels"

// selector is a restricted version of labels.Selector
type selector interface {
	Matches(labels.Labels) bool
}

// selectorDisjunction is the disjunction (logical or) of a list of selectors.
type selectorDisjunction []selector

func (d selectorDisjunction) Matches(labels2 labels.Labels) bool {
	for _, sel := range d {
		if sel.Matches(labels2) {
			return true
		}
	}
	return false
}

// or returns the logical or of the given selectors.
func or(sels ...selector) selector {
	return selectorDisjunction(sels)
}

// SelectorFromMap converts the given map to a selector. It correctly translates a `nil` map to a `nothing` collector,
// as is, e.g., used for headless services.
func SelectorFromMap(labelsMap map[string]string) labels.Selector {
	if len(labelsMap) == 0 {
		return labels.Nothing()
	}
	return labels.SelectorFromSet(labels.Set(labelsMap))
}
