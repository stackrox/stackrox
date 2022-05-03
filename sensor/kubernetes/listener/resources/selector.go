package resources

import (
	"k8s.io/apimachinery/pkg/labels"
)

// labelWithLen is label.Labels with added Len() function
type labelsWithLen interface {
	Has(label string) (exists bool)
	Get(label string) (value string)
	Len() uint
}

// selector is a restricted version of selectorWrap
type selector interface {
	Matches(labelsWithLen) bool
}

type labelWithLenImpl struct {
	labels map[string]string
}

func (l labelWithLenImpl) Has(label string) bool {
	_, exists := l.labels[label]
	return exists
}

func (l labelWithLenImpl) Get(label string) string {
	return l.labels[label]
}

func (l labelWithLenImpl) Len() uint {
	return uint(len(l.labels))
}

func createLabelsWithLen(labels map[string]string) labelWithLenImpl {
	return labelWithLenImpl{labels}
}

// selectorWrap holds a selector and information allowing for additional checks before matching
type selectorWrap struct {
	selector  labels.Selector
	numLabels uint
	matchNil  bool
}

func (s selectorWrap) Matches(labels labelsWithLen) bool {
	if s.numLabels > labels.Len() {
		return false
	}
	if s.numLabels == 0 {
		return s.matchNil
	}
	return s.selector.Matches(labels)
}

// selectorDisjunction is the disjunction (logical or) of a list of selectors.
type selectorDisjunction []selector

func (d selectorDisjunction) Matches(labels labelsWithLen) bool {
	for _, sel := range d {
		if sel.Matches(labels) {
			return true
		}
	}
	return false
}

// or returns the logical or of the given SelectorWrappers.
func or(sels ...selector) selector {
	return selectorDisjunction(sels)
}

type selectorWrapOption func(*selectorWrap)

// emptyMatchesNothing means that a set with no labels should not match with anything
func emptyMatchesNothing() selectorWrapOption {
	return func(sw *selectorWrap) {
		sw.matchNil = false
	}
}

// emptyMatchesEverything means that a set with no labels should match with everything
func emptyMatchesEverything() selectorWrapOption {
	return func(sw *selectorWrap) {
		sw.matchNil = true
	}
}

// createSelector returns a SelectorWrapper for the given map of labels; matchNil determines whether
// an empty set of labels matches everything or nothing.
func createSelector(labelsMap map[string]string, opts ...selectorWrapOption) selectorWrap {
	selWrapper := selectorWrap{matchNil: false}

	for _, opt := range opts {
		opt(&selWrapper)
	}

	selWrapper.numLabels = uint(len(labelsMap))
	if selWrapper.numLabels == 0 {
		if selWrapper.matchNil {
			selWrapper.selector = labels.Everything()
		} else {
			selWrapper.selector = labels.Nothing()
		}
		return selWrapper
	}
	selWrapper.selector = labels.SelectorFromSet(labels.Set(labelsMap))
	return selWrapper
}
