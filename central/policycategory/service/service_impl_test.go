package service

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/policycategory/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
)

func TestPolicyCategoryService(t *testing.T) {
	suite.Run(t, new(PolicyCategoryServiceTestSuite))
}

type PolicyCategoryServiceTestSuite struct {
	suite.Suite
	categories *mocks.MockDataStore
	tested     Service

	envIsolator *envisolator.EnvIsolator

	mockCtrl *gomock.Controller
}

func (s *PolicyCategoryServiceTestSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())

	s.mockCtrl = gomock.NewController(s.T())

	s.tested = New(
		s.categories,
	)
}

func (s *PolicyCategoryServiceTestSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
	s.mockCtrl.Finish()
}

func (s *PolicyCategoryServiceTestSuite) TestRenameInvalidNameFails() {
	ctx := context.Background()
	resp, err := s.tested.RenamePolicyCategory(ctx, &v1.NewRenamePolicyCategoryRequest{
		Id:              "id",
		NewCategoryName: "foo",
	})
	s.Nil(resp)
	s.Error(err)
	s.Equal(invalidNameErrString, err.Error())
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
	s.Equal(invalidNameErrString, err.Error())
}
