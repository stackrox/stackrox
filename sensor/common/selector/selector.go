package selector

import (
	"k8s.io/apimachinery/pkg/labels"
)

// labelWithLen is label.Labels with added Len() function
type labelsWithLen interface {
	Has(label string) (exists bool)
	Get(label string) (value string)
	Len() uint
}

// Selector is a restricted version of selectorWrap
type Selector interface {
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

func CreateLabelsWithLen(labels map[string]string) labelWithLenImpl {
	return labelWithLenImpl{labels}
}

// selectorWrap holds a Selector and information allowing for additional checks before matching
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

// selectorDisjunction is the disjunction (logical Or) of a list of selectors.
type selectorDisjunction []Selector

func (d selectorDisjunction) Matches(labels labelsWithLen) bool {
	for _, sel := range d {
		if sel.Matches(labels) {
			return true
		}
	}
	return false
}

// Or returns the logical Or of the given SelectorWrappers.
func Or(sels ...Selector) Selector {
	return selectorDisjunction(sels)
}

type selectorWrapOption func(*selectorWrap)

// EmptyMatchesNothing means that a set with no labels should not match with anything
func EmptyMatchesNothing() selectorWrapOption {
	return func(sw *selectorWrap) {
		sw.matchNil = false
	}
}

// EmptyMatchesEverything means that a set with no labels should match with everything
func EmptyMatchesEverything() selectorWrapOption {
	return func(sw *selectorWrap) {
		sw.matchNil = true
	}
}

// CreateSelector returns a SelectorWrapper for the given map of labels; matchNil determines whether
// an empty set of labels matches everything Or nothing.
func CreateSelector(labelsMap map[string]string, opts ...selectorWrapOption) selectorWrap {
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
