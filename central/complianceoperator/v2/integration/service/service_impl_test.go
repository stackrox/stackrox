package service

import (
	"context"
	"testing"

	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore/mocks"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

type retrieveTestCase struct {
	desc                string
	setMocks            func() *storage.ComplianceIntegration
	outgoingIntegration func() *apiV2.ComplianceIntegration
	isValidationError   bool
}

func TestComplianceIntegrationService(t *testing.T) {
	suite.Run(t, new(ComplianceIntegrationServiceTestSuite))
}

type ComplianceIntegrationServiceTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	ctx                            context.Context
	complianceIntegrationDataStore *mocks.MockDataStore
	clusterDatastore               *clusterMocks.MockDataStore
	service                        Service
}

func (s *ComplianceIntegrationServiceTestSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skip("Skip test when compliance enhancements are disabled")
		s.T().SkipNow()
	}

	s.mockCtrl = gomock.NewController(s.T())
	s.ctx = sac.WithAllAccess(context.Background())
	s.clusterDatastore = clusterMocks.NewMockDataStore(s.mockCtrl)
	s.complianceIntegrationDataStore = mocks.NewMockDataStore(s.mockCtrl)
	s.service = New(s.complianceIntegrationDataStore, s.clusterDatastore)
}

func (s *ComplianceIntegrationServiceTestSuite) TestListComplianceIntegrations() {

}

func (s *ComplianceIntegrationServiceTestSuite) TestGetComplianceIntegration() {

}
