package service

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/policy/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/status"
)

var (
	mockRequestOneID = &v1.ExportPoliciesRequest{
		PolicyIds: []string{"Joseph Rules"},
	}
	mockRequestTwoIDs = &v1.ExportPoliciesRequest{
		PolicyIds: []string{"Joseph Rules", "abcd"},
	}
)

func TestPolicyService(t *testing.T) {
	suite.Run(t, new(PolicyServiceTestSuite))
}

type PolicyServiceTestSuite struct {
	suite.Suite
	policies *mocks.MockDataStore
	tested   Service

	envIsolator *testutils.EnvIsolator

	mockCtrl *gomock.Controller
}

func (s *PolicyServiceTestSuite) SetupTest() {
	s.envIsolator = testutils.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(features.PolicyImportExport.EnvVar(), "true")

	s.mockCtrl = gomock.NewController(s.T())

	s.policies = mocks.NewMockDataStore(s.mockCtrl)

	s.tested = New(
		s.policies,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
}

func (s *PolicyServiceTestSuite) TearDownTest() {
	defer s.envIsolator.RestoreAll()
	s.mockCtrl.Finish()
}

func (s *PolicyServiceTestSuite) compareErrorsToExpected(expectedErrors []*v1.ExportPolicyError, apiError error) {
	apiStatus, ok := status.FromError(apiError)
	s.Require().True(ok)
	details := apiStatus.Details()
	s.Len(details, 1)
	exportErrors, ok := details[0].(*v1.ExportPoliciesErrorList)
	s.Require().True(ok)
	// actual errors == expected errors ignoring order
	s.Len(exportErrors.GetErrors(), len(expectedErrors))
	for _, expected := range expectedErrors {
		s.Contains(exportErrors.GetErrors(), expected)
	}
}

func makeError(errorID, errorString string) *v1.ExportPolicyError {
	return &v1.ExportPolicyError{
		PolicyId: errorID,
		Error: &v1.PolicyError{
			Error: errorString,
		},
	}
}

func (s *PolicyServiceTestSuite) TestExportInvalidIDFails() {
	ctx := context.Background()
	mockErrors := []*v1.ExportPolicyError{
		makeError(mockRequestOneID.PolicyIds[0], "not found"),
	}
	s.policies.EXPECT().GetPolicies(ctx, mockRequestOneID.PolicyIds).Return(make([]*storage.Policy, 0), []int{0}, []error{errors.New("not found")}, nil)
	resp, err := s.tested.ExportPolicies(ctx, mockRequestOneID)
	s.Nil(resp)
	s.Error(err)
	s.compareErrorsToExpected(mockErrors, err)
}

func (s *PolicyServiceTestSuite) TestExportValidIDSucceeds() {
	ctx := context.Background()
	mockPolicy := &storage.Policy{
		Id: mockRequestOneID.PolicyIds[0],
	}
	s.policies.EXPECT().GetPolicies(ctx, mockRequestOneID.PolicyIds).Return([]*storage.Policy{mockPolicy}, nil, nil, nil)
	resp, err := s.tested.ExportPolicies(ctx, mockRequestOneID)
	s.NoError(err)
	s.NotNil(resp)
	s.Len(resp.GetPolicies(), 1)
	s.Equal(mockPolicy, resp.Policies[0])
}

func (s *PolicyServiceTestSuite) TestExportMixedSuccessAndMissing() {
	ctx := context.Background()
	mockPolicy := &storage.Policy{
		Id: mockRequestTwoIDs.PolicyIds[0],
	}
	mockErrors := []*v1.ExportPolicyError{
		makeError(mockRequestTwoIDs.PolicyIds[1], "not found"),
	}
	s.policies.EXPECT().GetPolicies(ctx, mockRequestTwoIDs.PolicyIds).Return([]*storage.Policy{mockPolicy}, []int{1}, []error{errors.New("not found")}, nil)
	resp, err := s.tested.ExportPolicies(ctx, mockRequestTwoIDs)
	s.Nil(resp)
	s.Error(err)
	s.compareErrorsToExpected(mockErrors, err)
}

func (s *PolicyServiceTestSuite) TestMultipleFailures() {
	ctx := context.Background()
	errString := "test"
	storeErrors := []error{errors.New(errString), errors.New("not found")}
	mockErrors := []*v1.ExportPolicyError{
		makeError(mockRequestTwoIDs.PolicyIds[0], errString),
		makeError(mockRequestTwoIDs.PolicyIds[1], "not found"),
	}
	s.policies.EXPECT().GetPolicies(ctx, mockRequestTwoIDs.PolicyIds).Return(make([]*storage.Policy, 0), []int{0, 1}, storeErrors, nil)
	resp, err := s.tested.ExportPolicies(ctx, mockRequestTwoIDs)
	s.Nil(resp)
	s.Error(err)
	s.compareErrorsToExpected(mockErrors, err)
}
