package resources

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

func (m mockSelector) Add(r ...labels.Requirement) labels.Selector {
	return nil
}

func (m mockSelector) Requirements() (requirements labels.Requirements, selectable bool) {
	return nil, false
}

func (m mockSelector) DeepCopySelector() labels.Selector {
	return nil
}

func (m mockSelector) RequiresExactMatch(label string) (value string, found bool) {
	return "", false
}

func (m mockSelector) Matches(labels labels.Labels) bool {
	m.testSuite.hasMatchesBeenCalled = true
	return m.internalSelector.Matches(labels)
}

func (s *SelectorWrapperTestSuite) injectMockSelector(sw *selectorWrap) {
	sw.selector = mockSelector{sw.selector, s}
}

func (s *SelectorWrapperTestSuite) TestLabelMatching() {
	tests := map[string]struct {
		givenSelectorLabels                 map[string]string
		matchEmptySelector                  selectorWrapOption
		givenMatchingLabels                 map[string]string
		expectedMatch                       bool
		expectedSelectorMatchesFunctionCall bool
	}{
		"Empty selector with matchEmpty set to false should match nothing; attempting to match some labels": {
			givenSelectorLabels:                 labelsEmpty,
			matchEmptySelector:                  EmptyMatchesNothing(),
			givenMatchingLabels:                 labelsThreeElements,
			expectedMatch:                       false,
			expectedSelectorMatchesFunctionCall: false,
		},
		"Empty selector with matchEmpty set to false should match nothing; attempting to match empty labels": {
			givenSelectorLabels:                 labelsEmpty,
			matchEmptySelector:                  EmptyMatchesNothing(),
			givenMatchingLabels:                 labelsEmpty,
			expectedMatch:                       false,
			expectedSelectorMatchesFunctionCall: false,
		},
		"Empty selector with matchEmpty set to true should match everything; attempting to match some labels": {
			givenSelectorLabels:                 labelsEmpty,
			matchEmptySelector:                  EmptyMatchesEverything(),
			givenMatchingLabels:                 labelsFiveElements,
			expectedMatch:                       true,
			expectedSelectorMatchesFunctionCall: false,
		},
		"Empty selector with matchEmpty set to true should match everything; attempting to match empty labels": {
			givenSelectorLabels:                 labelsEmpty,
			matchEmptySelector:                  EmptyMatchesEverything(),
			givenMatchingLabels:                 labelsEmpty,
			expectedMatch:                       true,
			expectedSelectorMatchesFunctionCall: false,
		},
		"More selector labels than received labels -> no match and selector Matches function not called": {
			givenSelectorLabels:                 labelsThreeElements,
			matchEmptySelector:                  EmptyMatchesEverything(),
			givenMatchingLabels:                 labelsOneElement,
			expectedMatch:                       false,
			expectedSelectorMatchesFunctionCall: false,
		},
		"Equal number but different labels, selector Matches function should be called and return false": {
			givenSelectorLabels:                 labelsThreeElements,
			matchEmptySelector:                  EmptyMatchesEverything(),
			givenMatchingLabels:                 labelsThreeElements2,
			expectedMatch:                       false,
			expectedSelectorMatchesFunctionCall: true,
		},
		"Equal labels, selector Matches function should be called and return true": {
			givenSelectorLabels:                 labelsThreeElements,
			matchEmptySelector:                  EmptyMatchesEverything(),
			givenMatchingLabels:                 labelsThreeElements,
			expectedMatch:                       true,
			expectedSelectorMatchesFunctionCall: true,
		},
		"selector with one label, match with three labels including the one. Expected to return true and call Matches": {
			givenSelectorLabels:                 labelsOneElement,
			matchEmptySelector:                  EmptyMatchesEverything(),
			givenMatchingLabels:                 labelsThreeElements,
			expectedMatch:                       true,
			expectedSelectorMatchesFunctionCall: true,
		},
	}
	for name, tt := range tests {
		s.Run(name, func() {
			s.hasMatchesBeenCalled = false
			selectorWrap := createSelector(tt.givenSelectorLabels, tt.matchEmptySelector)
			s.injectMockSelector(&selectorWrap)

			receivedMatch := selectorWrap.Matches(createLabelsWithLen(tt.givenMatchingLabels))
			receivedSelectorMatchesFunctionCalled := s.hasMatchesBeenCalled

			s.Equal(tt.expectedMatch, receivedMatch)
			s.Equal(tt.expectedSelectorMatchesFunctionCall, receivedSelectorMatchesFunctionCalled)
		})
	}
}

func (s *SelectorWrapperTestSuite) TestLabelMatchingWithDisjunctions() {
	tests := map[string]struct {
		givenSelectorLabels                 []map[string]string
		matchEmptySelector                  []selectorWrapOption
		givenMatchingLabels                 map[string]string
		expectedMatch                       bool
		expectedSelectorMatchesFunctionCall bool
	}{
		"Disjunction of empty list that should match nothing and labels with three elements": {
			givenSelectorLabels:                 []map[string]string{labelsEmpty, labelsThreeElements},
			matchEmptySelector:                  []selectorWrapOption{EmptyMatchesNothing(), EmptyMatchesEverything()},
			givenMatchingLabels:                 labelsOneElement,
			expectedMatch:                       false,
			expectedSelectorMatchesFunctionCall: false,
		},
		"Disjunction of empty list that should match everything and labels with three elements": {
			givenSelectorLabels:                 []map[string]string{labelsEmpty, labelsThreeElements},
			matchEmptySelector:                  []selectorWrapOption{EmptyMatchesEverything(), EmptyMatchesEverything()},
			givenMatchingLabels:                 labelsOneElement,
			expectedMatch:                       true,
			expectedSelectorMatchesFunctionCall: false,
		},
		"Disjunction of two selectors with more labels than the input": {
			givenSelectorLabels:                 []map[string]string{labelsFiveElements, labelsThreeElements},
			matchEmptySelector:                  []selectorWrapOption{EmptyMatchesNothing(), EmptyMatchesNothing()},
			givenMatchingLabels:                 labelsOneElement,
			expectedMatch:                       false,
			expectedSelectorMatchesFunctionCall: false,
		},
		"Disjunction of two selectors, one more labels than the input, one equal, without match": {
			givenSelectorLabels:                 []map[string]string{labelsFiveElements, labelsThreeElements},
			matchEmptySelector:                  []selectorWrapOption{EmptyMatchesNothing(), EmptyMatchesNothing()},
			givenMatchingLabels:                 labelsThreeElements2,
			expectedMatch:                       false,
			expectedSelectorMatchesFunctionCall: true,
		},
		"Disjunction of two selectors, one more labels than the input, one equal, with match": {
			givenSelectorLabels:                 []map[string]string{labelsFiveElements, labelsThreeElements},
			matchEmptySelector:                  []selectorWrapOption{EmptyMatchesNothing(), EmptyMatchesNothing()},
			givenMatchingLabels:                 labelsThreeElements,
			expectedMatch:                       true,
			expectedSelectorMatchesFunctionCall: true,
		},
		"Disjunction of two selectors, with less labels than the input": {
			givenSelectorLabels:                 []map[string]string{labelsThreeElements2, labelsThreeElements},
			matchEmptySelector:                  []selectorWrapOption{EmptyMatchesNothing(), EmptyMatchesNothing()},
			givenMatchingLabels:                 labelsFiveElements,
			expectedMatch:                       false,
			expectedSelectorMatchesFunctionCall: true,
		},
	}
	for name, tt := range tests {
		s.Run(name, func() {
			s.hasMatchesBeenCalled = false
			var selectorWrappers []selector
			for i := 0; i < len(tt.givenSelectorLabels); i++ {
				newSelector := createSelector(tt.givenSelectorLabels[i], tt.matchEmptySelector[i])
				s.injectMockSelector(&newSelector)
				selectorWrappers = append(selectorWrappers, newSelector)
			}

			sel := or(selectorWrappers...)

			receivedMatch := sel.Matches(createLabelsWithLen(tt.givenMatchingLabels))
			receivedSelectorMatchesFunctionCalled := s.hasMatchesBeenCalled

			s.Equal(tt.expectedMatch, receivedMatch)
			s.Equal(tt.expectedSelectorMatchesFunctionCall, receivedSelectorMatchesFunctionCalled)
		})
	}
}
