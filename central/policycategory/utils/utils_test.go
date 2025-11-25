package utils

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
)

func TestPolicyCategoryUtils(t *testing.T) {
	suite.Run(t, new(PolicyCategoryUtilsTestSuite))
}

type PolicyCategoryUtilsTestSuite struct {
	suite.Suite
}

func (s *PolicyCategoryUtilsTestSuite) SetupTest() {
}

func (s *PolicyCategoryUtilsTestSuite) TestGetCategoryNamesToIDs() {
	categories := []*storage.PolicyCategory{
		{Id: "c1", Name: "Category 1"},
		{Id: "c3", Name: "CATegory 1"},
		{Id: "cc1", Name: "CaTeGory 2"},
		{Id: "cc2", Name: "Category 2"},
		{Id: "ccc1", Name: "Category 3"},
		{Id: "ccc2", Name: "CaTegory 3"},
	}
	expected := map[string]string{
		"Category 1": "c3",
		"CATegory 1": "c3",
		"CaTeGory 2": "cc1",
		"Category 2": "cc1",
		"Category 3": "ccc2",
		"CaTegory 3": "ccc2",
	}
	result := GetCategoryNameToIDs(categories)
	s.Equal(expected, result)
}
