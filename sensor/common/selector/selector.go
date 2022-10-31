package selector

import (
	"k8s.io/apimachinery/pkg/labels"
)

// LabelsWithLen is label.Labels with added Len() function
type LabelsWithLen interface {
	Has(label string) (exists bool)
	Get(label string) (value string)
	Len() uint
}

// Selector is a restricted version of wrap
type Selector interface {
	Matches(LabelsWithLen) bool
}

type labelWithLenImpl struct {
	labels map[string]string
}

// Has label
func (l labelWithLenImpl) Has(label string) bool {
	_, exists := l.labels[label]
	return exists
}

// Get label
func (l labelWithLenImpl) Get(label string) string {
	return l.labels[label]
}

// Len returns length of labels
func (l labelWithLenImpl) Len() uint {
	return uint(len(l.labels))
}

// CreateLabelsWithLen create labels wrapper that respects the LabelsWithLen interface
func CreateLabelsWithLen(labels map[string]string) LabelsWithLen {
	return labelWithLenImpl{labels}
}

// wrap holds a selector and information allowing for additional checks before matching
type wrap struct {
	selector  labels.Selector
	numLabels uint
	matchNil  bool
}

// Matches a set of labels
func (s wrap) Matches(labels LabelsWithLen) bool {
	if s.numLabels > labels.Len() {
		return false
	}
	if s.numLabels == 0 {
		return s.matchNil
	}
	return s.selector.Matches(labels)
}

// selectorDisjunction is the disjunction (logical or) of a list of selectors.
type selectorDisjunction []Selector

// Matches a set of labels
func (d selectorDisjunction) Matches(labels LabelsWithLen) bool {
	for _, sel := range d {
		if sel.Matches(labels) {
			return true
		}
	}
	return false
}

// Or returns the logical or of the given SelectorWrappers.
func Or(sels ...Selector) Selector {
	return selectorDisjunction(sels)
}

// Options function interface to define properties of selectors
type Options func(*wrap)

// EmptyMatchesNothing means that a set with no labels should not match with anything
func EmptyMatchesNothing() Options {
	return func(sw *wrap) {
		sw.matchNil = false
	}
}

// EmptyMatchesEverything means that a set with no labels should match with everything
func EmptyMatchesEverything() Options {
	return func(sw *wrap) {
		sw.matchNil = true
	}
}

// CreateSelector returns a SelectorWrapper for the given map of labels; matchNil determines whether
// an empty set of labels matches everything or nothing.
func CreateSelector(labelsMap map[string]string, opts ...Options) Selector {
	selWrapper := &wrap{matchNil: false}

	for _, opt := range opts {
		opt(selWrapper)
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
