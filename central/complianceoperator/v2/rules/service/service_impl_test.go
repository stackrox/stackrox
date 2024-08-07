package service

import (
	"context"
	"testing"

	ruleMocks "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore/mocks"
	convertUtils "github.com/stackrox/rox/central/convert/testutils"
	v1 "github.com/stackrox/rox/generated/api/v1"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestComplianceRuleService(t *testing.T) {
	suite.Run(t, new(ComplianceRulesServiceTestSuite))
}

type ComplianceRulesServiceTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	ctx           context.Context
	ruleDatastore *ruleMocks.MockDataStore
	service       Service
}

func (s *ComplianceRulesServiceTestSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skip("Skip test when compliance enhancements are disabled")
		s.T().SkipNow()
	}

	s.ctx = sac.WithAllAccess(context.Background())
}

func (s *ComplianceRulesServiceTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.ruleDatastore = ruleMocks.NewMockDataStore(s.mockCtrl)

	s.service = New(s.ruleDatastore)
}

func (s *ComplianceRulesServiceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ComplianceRulesServiceTestSuite) TestGetComplianceRuleByName() {
	ruleName := "ocp-cis-4.2"
	query := search.ConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorRuleName, ruleName).ProtoQuery(),
		search.EmptyQuery(),
	)
	query.Pagination = &v1.QueryPagination{Limit: maxPaginationLimit}
	s.ruleDatastore.EXPECT().SearchRules(s.ctx, query).Return([]*storage.ComplianceOperatorRuleV2{convertUtils.GetRuleV2Storage(s.T())}, nil)

	rule, err := s.service.GetComplianceRule(s.ctx, &apiV2.RuleRequest{RuleName: ruleName})
	s.Require().NoError(err)
	protoassert.Equal(s.T(), convertUtils.GetRuleV2(s.T()), rule)
}

func (s *ComplianceRulesServiceTestSuite) TestGetComplianceRuleByNameNotPresent() {
	rule, err := s.service.GetComplianceRule(s.ctx, &apiV2.RuleRequest{})
	s.Require().Error(err)
	s.Require().Empty(rule)
}
