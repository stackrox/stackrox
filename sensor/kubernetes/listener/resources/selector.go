package resources

import (
	"math"

	"k8s.io/apimachinery/pkg/labels"
)

// SelectorWrapper holds a selector and information allowing for additional checks before matching
type selectorWrapper struct {
	selector  selector
	numLabels uint
	matchNil  bool
}

func (s *selectorWrapper) getSelector() selector {
	return s.selector
}

func (s *selectorWrapper) matches(labels labels.Labels, numLabels uint) bool {
	if s.numLabels > numLabels {
		return false
	}
	if s.numLabels == 0 {
		return s.matchNil
	}
	return s.selector.Matches(labels)
}

// selector is a restricted version of labels.Selector
type selector interface {
	Matches(labels.Labels) bool
}

// selectorDisjunction is the disjunction (logical or) of a list of selectors.
type selectorDisjunction []selector

func (d selectorDisjunction) Matches(labels labels.Labels) bool {
	for _, sel := range d {
		if sel.Matches(labels) {
			return true
		}
	}
	return false
}

// or returns the logical or of the given SelectorWrappers.
func or(sels ...selectorWrapper) selectorWrapper {
	var selWrapper = selectorWrapper{nil, math.MaxUint, false}
	var selectors selectorDisjunction
	for _, s := range sels {
		if s.matchNil {
			selWrapper.matchNil = true
		}
		if selWrapper.numLabels > s.numLabels && (s.numLabels > 0 || s.matchNil) {
			selWrapper.numLabels = s.numLabels
		}
		selectors = append(selectors, s.selector)
	}
	if selWrapper.numLabels == math.MaxUint {
		selWrapper.numLabels = 0
	}
	selWrapper.selector = selectors
	return selWrapper
}

// CreateSelector returns a SelectorWrapper for the given map of labels; matchNil determines whether
// an empty set of labels matches everything or nothing.
func createSelector(labelsMap map[string]string, matchNil bool) selectorWrapper {
	var selWrapper selectorWrapper
	selWrapper.numLabels = uint(len(labelsMap))
	if matchNil {
		selWrapper.matchNil = true
		if selWrapper.numLabels == 0 {
			selWrapper.selector = labels.Everything()
		}
	} else {
		selWrapper.matchNil = false
		if selWrapper.numLabels == 0 {
			selWrapper.selector = labels.Nothing()
		}
	}
	selWrapper.selector = labels.SelectorFromSet(labels.Set(labelsMap))
	return selWrapper
}
