package selector

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/labels"
)

var (
	labelsEmpty          = map[string]string{}
	labelsOneElement     = map[string]string{"FirstEle": "FirstEleValue"}
	labelsThreeElements  = map[string]string{"FirstEle": "FirstEleValue", "2nd": "2ndValue", "3rd": "3rdValue"}
	labelsThreeElements2 = map[string]string{"4th": "Val4", "5th": "Val5", "6th": "Val6"}
	labelsFiveElements   = map[string]string{"1": "2", "2": "3", "3": "4", "4": "5", "5": "6"}
)

var _ suite.SetupTestSuite = (*SelectorWrapperTestSuite)(nil)

func TestSelectorWrapper(t *testing.T) {
	suite.Run(t, new(SelectorWrapperTestSuite))
}

func (s *SelectorWrapperTestSuite) SetupTest() {
	s.hasMatchesBeenCalled = false
}

type SelectorWrapperTestSuite struct {
	suite.Suite
	hasMatchesBeenCalled bool
}

type mockSelector struct {
	internalSelector labels.Selector
	testSuite        *SelectorWrapperTestSuite
}

func (m mockSelector) Empty() bool {
	return false
}

func (m mockSelector) String() string {
	return ""
}

func (m mockSelector) Add(_ ...labels.Requirement) labels.Selector {
	return nil
}

func (m mockSelector) Requirements() (requirements labels.Requirements, selectable bool) {
	return nil, false
}

func (m mockSelector) DeepCopySelector() labels.Selector {
	return nil
}

func (m mockSelector) RequiresExactMatch(_ string) (value string, found bool) {
	return "", false
}

func (m mockSelector) Matches(labels labels.Labels) bool {
	m.testSuite.hasMatchesBeenCalled = true
	return m.internalSelector.Matches(labels)
}

func (s *SelectorWrapperTestSuite) injectMockSelector(sw *wrap) {
	sw.selector = mockSelector{sw.selector, s}
}

func (s *SelectorWrapperTestSuite) TestLabelMatching() {
	tests := map[string]struct {
		givenSelectorLabels                map[string]string
		matchEmptySelector                 Options
		givenMatchingLabels                map[string]string
		expectedMatch                      bool
		expectedMatchesInsideMatchesCalled bool
	}{
		"Empty selector with matchEmpty set to false should match nothing; attempting to match some labels": {
			givenSelectorLabels:                labelsEmpty,
			matchEmptySelector:                 EmptyMatchesNothing(),
			givenMatchingLabels:                labelsThreeElements,
			expectedMatch:                      false,
			expectedMatchesInsideMatchesCalled: false,
		},
		"Empty selector with matchEmpty set to false should match nothing; attempting to match empty labels": {
			givenSelectorLabels:                labelsEmpty,
			matchEmptySelector:                 EmptyMatchesNothing(),
			givenMatchingLabels:                labelsEmpty,
			expectedMatch:                      false,
			expectedMatchesInsideMatchesCalled: false,
		},
		"Empty selector with matchEmpty set to true should match everything; attempting to match some labels": {
			givenSelectorLabels:                labelsEmpty,
			matchEmptySelector:                 EmptyMatchesEverything(),
			givenMatchingLabels:                labelsFiveElements,
			expectedMatch:                      true,
			expectedMatchesInsideMatchesCalled: false,
		},
		"Empty selector with matchEmpty set to true should match everything; attempting to match empty labels": {
			givenSelectorLabels:                labelsEmpty,
			matchEmptySelector:                 EmptyMatchesEverything(),
			givenMatchingLabels:                labelsEmpty,
			expectedMatch:                      true,
			expectedMatchesInsideMatchesCalled: false,
		},
		"More selector labels than received labels -> no match and selector Matches function not called": {
			givenSelectorLabels:                labelsThreeElements,
			matchEmptySelector:                 EmptyMatchesEverything(),
			givenMatchingLabels:                labelsOneElement,
			expectedMatch:                      false,
			expectedMatchesInsideMatchesCalled: false,
		},
		"Equal number but different labels, selector Matches function should be called and return false": {
			givenSelectorLabels:                labelsThreeElements,
			matchEmptySelector:                 EmptyMatchesEverything(),
			givenMatchingLabels:                labelsThreeElements2,
			expectedMatch:                      false,
			expectedMatchesInsideMatchesCalled: true,
		},
		"Equal labels, selector Matches function should be called and return true": {
			givenSelectorLabels:                labelsThreeElements,
			matchEmptySelector:                 EmptyMatchesEverything(),
			givenMatchingLabels:                labelsThreeElements,
			expectedMatch:                      true,
			expectedMatchesInsideMatchesCalled: true,
		},
		"selector with one label, match with three labels including the one. Expected to return true and call Matches": {
			givenSelectorLabels:                labelsOneElement,
			matchEmptySelector:                 EmptyMatchesEverything(),
			givenMatchingLabels:                labelsThreeElements,
			expectedMatch:                      true,
			expectedMatchesInsideMatchesCalled: true,
		},
	}
	for name, tt := range tests {
		s.Run(name, func() {
			s.hasMatchesBeenCalled = false
			selectorWrap := CreateSelector(tt.givenSelectorLabels, tt.matchEmptySelector)
			wrapObj, ok := selectorWrap.(*wrap)
			s.Require().True(ok, "return must be a wrap object")
			s.injectMockSelector(wrapObj)

			s.Equal(tt.expectedMatch, selectorWrap.Matches(CreateLabelsWithLen(tt.givenMatchingLabels)))
			s.Equal(tt.expectedMatchesInsideMatchesCalled, s.hasMatchesBeenCalled)
		})
	}
}

func (s *SelectorWrapperTestSuite) TestLabelMatchingWithDisjunctions() {
	tests := map[string]struct {
		givenSelectorLabels                []map[string]string
		matchEmptySelector                 []Options
		givenMatchingLabels                map[string]string
		expectedMatch                      bool
		expectedMatchesInsideMatchesCalled bool
	}{
		"Disjunction of empty list that should match nothing and labels with three elements": {
			givenSelectorLabels:                []map[string]string{labelsEmpty, labelsThreeElements},
			matchEmptySelector:                 []Options{EmptyMatchesNothing(), EmptyMatchesEverything()},
			givenMatchingLabels:                labelsOneElement,
			expectedMatch:                      false,
			expectedMatchesInsideMatchesCalled: false,
		},
		"Disjunction of empty list that should match everything and labels with three elements": {
			givenSelectorLabels:                []map[string]string{labelsEmpty, labelsThreeElements},
			matchEmptySelector:                 []Options{EmptyMatchesEverything(), EmptyMatchesEverything()},
			givenMatchingLabels:                labelsOneElement,
			expectedMatch:                      true,
			expectedMatchesInsideMatchesCalled: false,
		},
		"Disjunction of two selectors with more labels than the input": {
			givenSelectorLabels:                []map[string]string{labelsFiveElements, labelsThreeElements},
			matchEmptySelector:                 []Options{EmptyMatchesNothing(), EmptyMatchesNothing()},
			givenMatchingLabels:                labelsOneElement,
			expectedMatch:                      false,
			expectedMatchesInsideMatchesCalled: false,
		},
		"Disjunction of two selectors, one more labels than the input, one equal, without match": {
			givenSelectorLabels:                []map[string]string{labelsFiveElements, labelsThreeElements},
			matchEmptySelector:                 []Options{EmptyMatchesNothing(), EmptyMatchesNothing()},
			givenMatchingLabels:                labelsThreeElements2,
			expectedMatch:                      false,
			expectedMatchesInsideMatchesCalled: true,
		},
		"Disjunction of two selectors, one more labels than the input, one equal, with match": {
			givenSelectorLabels:                []map[string]string{labelsFiveElements, labelsThreeElements},
			matchEmptySelector:                 []Options{EmptyMatchesNothing(), EmptyMatchesNothing()},
			givenMatchingLabels:                labelsThreeElements,
			expectedMatch:                      true,
			expectedMatchesInsideMatchesCalled: true,
		},
		"Disjunction of two selectors, with less labels than the input": {
			givenSelectorLabels:                []map[string]string{labelsThreeElements2, labelsThreeElements},
			matchEmptySelector:                 []Options{EmptyMatchesNothing(), EmptyMatchesNothing()},
			givenMatchingLabels:                labelsFiveElements,
			expectedMatch:                      false,
			expectedMatchesInsideMatchesCalled: true,
		},
	}
	for name, tt := range tests {
		s.Run(name, func() {
			s.hasMatchesBeenCalled = false
			var selectorWrappers []Selector
			for i, label := range tt.givenSelectorLabels {
				newSelector := CreateSelector(label, tt.matchEmptySelector[i])
				wrapObj, ok := newSelector.(*wrap)
				s.Require().True(ok, "return must be a wrap object")

				s.injectMockSelector(wrapObj)
				selectorWrappers = append(selectorWrappers, newSelector)
			}

			sel := Or(selectorWrappers...)

			s.Equal(tt.expectedMatch, sel.Matches(CreateLabelsWithLen(tt.givenMatchingLabels)))
			s.Equal(tt.expectedMatchesInsideMatchesCalled, s.hasMatchesBeenCalled)
		})
	}
}

func (s *SelectorWrapperTestSuite) TestLabelWithLenImpl() {
	tests := map[string]struct {
		labels map[string]string
	}{
		"empty labels": {
			labels: map[string]string{},
		},
		"single label": {
			labels: map[string]string{"key1": "value1"},
		},
		"multiple labels": {
			labels: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
		},
		"labels with empty values": {
			labels: map[string]string{
				"key1": "",
				"key2": "value2",
			},
		},
		"labels with special characters": {
			labels: map[string]string{
				"app.kubernetes.io/name": "myapp",
				"version":                "1.0.0",
				"env":                    "prod",
			},
		},
	}

	for name, tt := range tests {
		s.Run(name, func() {
			labelsWithLen := CreateLabelsWithLen(tt.labels)
			impl, ok := labelsWithLen.(labelWithLenImpl)
			s.Require().True(ok, "CreateLabelsWithLen should return labelWithLenImpl")

			// Test Len method
			s.Equal(uint(len(tt.labels)), impl.Len(), "Len() should return correct length")

			// Test Has method for existing keys
			for key := range tt.labels {
				s.True(impl.Has(key), "Has() should return true for existing key: %s", key)
			}

			// Test Has method for non-existing keys
			s.False(impl.Has("nonexistent"), "Has() should return false for non-existing key")
			s.False(impl.Has(""), "Has() should return false for empty key when not present")

			// Test Get method for existing keys
			for key, expectedValue := range tt.labels {
				actualValue := impl.Get(key)
				s.Equal(expectedValue, actualValue, "Get() should return correct value for key: %s", key)
			}

			// Test Get method for non-existing keys
			s.Equal("", impl.Get("nonexistent"), "Get() should return empty string for non-existing key")

			// Test Lookup method for existing keys
			for key, expectedValue := range tt.labels {
				actualValue, exists := impl.Lookup(key)
				s.True(exists, "Lookup() should return exists=true for existing key: %s", key)
				s.Equal(expectedValue, actualValue, "Lookup() should return correct value for key: %s", key)
			}

			// Test Lookup method for non-existing keys
			value, exists := impl.Lookup("nonexistent")
			s.False(exists, "Lookup() should return exists=false for non-existing key")
			s.Equal("", value, "Lookup() should return empty string for non-existing key")
		})
	}
}

func (s *SelectorWrapperTestSuite) TestLabelWithLenImplEdgeCases() {
	s.Run("nil map handling", func() {
		// Test that CreateLabelsWithLen handles nil map gracefully
		labelsWithLen := CreateLabelsWithLen(nil)
		impl, ok := labelsWithLen.(labelWithLenImpl)
		s.Require().True(ok, "CreateLabelsWithLen should return labelWithLenImpl even with nil map")

		s.Equal(uint(0), impl.Len(), "Len() should return 0 for nil map")
		s.False(impl.Has("any"), "Has() should return false for any key with nil map")
		s.Equal("", impl.Get("any"), "Get() should return empty string for any key with nil map")

		value, exists := impl.Lookup("any")
		s.False(exists, "Lookup() should return exists=false for any key with nil map")
		s.Equal("", value, "Lookup() should return empty string for any key with nil map")
	})

	s.Run("empty string key", func() {
		labelsWithLen := CreateLabelsWithLen(map[string]string{"": "empty-key-value", "normal": "normal-value"})
		impl, ok := labelsWithLen.(labelWithLenImpl)
		s.Require().True(ok)

		// Test that empty string key is handled correctly
		s.True(impl.Has(""), "Has() should return true for empty string key when it exists")
		s.Equal("empty-key-value", impl.Get(""), "Get() should return correct value for empty string key")

		value, exists := impl.Lookup("")
		s.True(exists, "Lookup() should return exists=true for empty string key when it exists")
		s.Equal("empty-key-value", value, "Lookup() should return correct value for empty string key")
	})
}
