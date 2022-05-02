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

// selector is a restricted version of selectorWrap
type selector interface {
	Matches(labelsWithLen) bool
	getSelector() labels.Selector
	getNumLabels() uint
	getMatchNil() bool
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

// SelectorWrap holds a selector and information allowing for additional checks before matching
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

func (s selectorWrap) getSelector() labels.Selector {
	return s.selector
}

func (s selectorWrap) getNumLabels() uint {
	return s.numLabels
}

func (s selectorWrap) getMatchNil() bool {
	return s.matchNil
}

// selectorDisjunction is the disjunction (logical or) of a list of selectors.
type selectorDisjunction []labels.Selector

func (d selectorDisjunction) Empty() bool {
	for _, sel := range d {
		if !sel.Empty() {
			return false
		}
	}
	return true
}

func (d selectorDisjunction) String() string {
	//TODO implement me
	panic("implement me")
}

func (d selectorDisjunction) Add(r ...labels.Requirement) labels.Selector {
	//TODO implement me
	panic("implement me")
}

func (d selectorDisjunction) Requirements() (requirements labels.Requirements, selectable bool) {
	//TODO implement me
	panic("implement me")
}

func (d selectorDisjunction) DeepCopySelector() labels.Selector {
	//TODO implement me
	panic("implement me")
}

func (d selectorDisjunction) RequiresExactMatch(label string) (value string, found bool) {
	//TODO implement me
	panic("implement me")
}

func (d selectorDisjunction) Matches(labels labels.Labels) bool {
	for _, sel := range d {
		if sel.Matches(labels) {
			return true
		}
	}
	return false
}

// or returns the logical or of the given SelectorWrappers.
func or(sels ...selector) selector {
	var selWrapper = selectorWrap{nil, math.MaxUint, false}
	var selectors selectorDisjunction
	for _, s := range sels {
		if s.getMatchNil() {
			selWrapper.matchNil = true
		}
		if selWrapper.numLabels > s.getNumLabels() && (s.getNumLabels() > 0 || s.getMatchNil()) {
			selWrapper.numLabels = s.getNumLabels()
		}
		selectors = append(selectors, s.getSelector())
	}
	if selWrapper.numLabels == math.MaxUint {
		selWrapper.numLabels = 0
	}
	selWrapper.selector = selectors
	return selWrapper
}

// CreateSelector returns a SelectorWrapper for the given map of labels; matchNil determines whether
// an empty set of labels matches everything or nothing.
func createSelector(labelsMap map[string]string, matchNil bool) selectorWrap {
	var selWrapper selectorWrap
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
