package detection

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
)

var (
	testCategories = []string{"Joseph rules"}
	testPolicyOne  = &storage.Policy{
		Categories: testCategories,
	}
	testPolicyTwo = &storage.Policy{
		Categories: []string{
			"anything",
			"at",
			"all",
		},
	}
)

func TestDetectorImpl(t *testing.T) {
	suite.Run(t, new(DetectorImplTestSuite))
}

type DetectorImplTestSuite struct {
	suite.Suite
}

func (s *DetectorImplTestSuite) SetupTest() {

}

func (s *DetectorImplTestSuite) TestHasAllowedCategories() {
	allowedCategoriesFilter, getUnusedCategories := MakeCategoryFilter(testCategories)
	allowed := allowedCategoriesFilter(testPolicyOne)
	s.True(allowed)
	s.Empty(getUnusedCategories())
}

func (s *DetectorImplTestSuite) TestNoAllowedCategories() {
	allowedCategoriesFilter, getUnusedCategories := MakeCategoryFilter(nil)
	allowed := allowedCategoriesFilter(testPolicyOne)
	s.True(allowed)
	s.Empty(getUnusedCategories())
}

func (s *DetectorImplTestSuite) TestDoesNotHaveAllowedCategories() {
	allowedCategoriesFilter, getUnusedCategories := MakeCategoryFilter(testCategories)
	allowed := allowedCategoriesFilter(testPolicyTwo)
	s.False(allowed)
	unusedCategories := getUnusedCategories()
	s.Len(unusedCategories, len(testCategories))
	for _, testCategory := range testCategories {
		s.Contains(unusedCategories, testCategory)
	}
}
