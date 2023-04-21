package service

import (
	"testing"

	"context"

	"github.com/golang/mock/gomock"
	storageMocks "github.com/stackrox/rox/central/compliance/datastore/mocks"
	standardsMocks "github.com/stackrox/rox/central/compliance/standards/mocks"
	managerMocks "github.com/stackrox/rox/central/complianceoperator/manager/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestNotifierService(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(complianceServiceTestSuite))
}

type complianceServiceTestSuite struct {
	suite.Suite

	ctrl      *gomock.Controller
	datastore *storageMocks.MockDataStore
	standards *standardsMocks.MockRepository
	manager   *managerMocks.MockManager

	ctx context.Context
}

func (s *complianceServiceTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.datastore = storageMocks.NewMockDataStore(s.ctrl)
	s.standards = standardsMocks.NewMockRepository(s.ctrl)
	s.manager = managerMocks.NewMockManager(s.ctrl)
	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())

}

func (s *complianceServiceTestSuite) getSvc() Service {
	return &serviceImpl{
		complianceDataStore: s.datastore,
		standardsRepo:       s.standards,
		manager:             s.manager,
	}
}

func getStandards() []*v1.ComplianceStandardMetadata {

	standards := []*v1.ComplianceStandardMetadata{
		&v1.ComplianceStandardMetadata{
			Name: "CIS_Docker_v1_2_0",
			Id:   "CIS_Docker_v1_2_0",
		},
	}

	return standards

}

func (s *complianceServiceTestSuite) TestGetStandards() {
	s.standards.EXPECT().Standards().Return(getStandards(), nil)
	s.manager.EXPECT().IsStandardActive(gomock.Any()).Return(true)

	standards, err := s.getSvc().GetStandards(s.ctx, &v1.Empty{})
	s.NoError(err)
	s.NotEmpty(standards)

}
