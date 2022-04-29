package resources

import (
	"math"

	"k8s.io/apimachinery/pkg/labels"
)

// labelWithLen is label.Labels with added Len() function
type labelsWithLen interface {
	Has(label string) (exists bool)
	Get(label string) (value string)
	Len() uint
}

// selector is a restricted version of selectorWrapper
type selector interface {
	Matches(labelsWithLen) bool
}

// internalSelector is a restricted version of labels.Selector
type internalSelector interface {
	Matches(labels.Labels) bool
}

type labelWrapper struct {
	labels    labels.Labels
	numLabels uint
}

func (l labelWrapper) Has(label string) bool {
	return l.labels.Has(label)
}

func (l labelWrapper) Get(label string) string {
	return l.labels.Get(label)
}

func (l labelWrapper) Len() uint {
	return l.numLabels
}

type restrictedSelector struct {
	selector labels.Selector
}

func (r restrictedSelector) Matches(labels labels.Labels) bool {
	return r.selector.Matches(labels)
}

// SelectorWrapper holds a selector and information allowing for additional checks before matching
type selectorWrapper struct {
	selector  internalSelector
	numLabels uint
	matchNil  bool
}

func (s *selectorWrapper) Matches(labels labelsWithLen) bool {
	if s.numLabels > labels.Len() {
		return false
	}
	if s.numLabels == 0 {
		return s.matchNil
	}
	return s.selector.Matches(labels)
}

// selectorDisjunction is the disjunction (logical or) of a list of selectors.
type selectorDisjunction []internalSelector

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
			selWrapper.selector = restrictedSelector{labels.Everything()}
		}
	} else {
		selWrapper.matchNil = false
		if selWrapper.numLabels == 0 {
			selWrapper.selector = restrictedSelector{labels.Nothing()}
		}
	}
	selWrapper.selector = restrictedSelector{labels.SelectorFromSet(labels.Set(labelsMap))}
	return selWrapper
}
