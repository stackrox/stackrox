package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/policycategory/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestPolicyCategoryService(t *testing.T) {
	suite.Run(t, new(PolicyCategoryServiceTestSuite))
}

type PolicyCategoryServiceTestSuite struct {
	suite.Suite
	categories *mocks.MockDataStore
	tested     Service

	mockCtrl *gomock.Controller
}

func (s *PolicyCategoryServiceTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.tested = New(
		s.categories,
	)
}

func (s *PolicyCategoryServiceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *PolicyCategoryServiceTestSuite) TestRenameInvalidNameFails() {
	ctx := context.Background()
	resp, err := s.tested.RenamePolicyCategory(ctx, &v1.RenamePolicyCategoryRequest{
		Id:              "id",
		NewCategoryName: "foo",
	})
	s.Nil(resp)
	s.Error(err)
	s.Equal(fmt.Sprintf("%s: %s", invalidNameErrString, errox.InvalidArgs.Error()), err.Error())

}

func (s *PolicyCategoryServiceTestSuite) TesPostInvalidNameFails() {
	ctx := context.Background()
	resp, err := s.tested.PostPolicyCategory(ctx, &v1.PostPolicyCategoryRequest{
		PolicyCategory: &v1.PolicyCategory{
			Id:        "id",
			Name:      " ",
			IsDefault: false,
		},
	})
	s.Nil(resp)
	s.Error(err)
	s.Equal(fmt.Sprintf("%s: %s", invalidNameErrString, errox.InvalidArgs.Error()), err.Error())
}
