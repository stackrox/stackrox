//go:build sql_integration

package service

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

func TestServiceImplWithDB_AccessScopes(t *testing.T) {
	suite.Run(t, new(serviceImplAccessScopeTestSuite))
}

type serviceImplAccessScopeTestSuite struct {
	suite.Suite

	tester *serviceImplTester
}

func (s *serviceImplAccessScopeTestSuite) SetupSuite() {
	s.tester = &serviceImplTester{}
	s.tester.Setup(s.T())
}

func (s *serviceImplAccessScopeTestSuite) SetupTest() {
	s.Require().NotNil(s.tester)
	s.tester.SetupTest(s.T())
}

func (s *serviceImplAccessScopeTestSuite) TearDownTest() {
	s.Require().NotNil(s.tester)
	s.tester.TearDownTest(s.T())
}

func (s *serviceImplAccessScopeTestSuite) TestListAccessScopes() {
	t := s.T()
	ctx := sac.WithAllAccess(t.Context())

	accessScopeName1 := "TestListAccessScopes_noTraits"
	accessScopeName2 := "TestListAccessScopes_imperativeOriginTraits"
	accessScopeName3 := "TestListAccessScopes_declarativeOriginTraits"
	accessScopeName4 := "TestListAccessScopes_orphanedDeclarativeOriginTraits"
	accessScopeName5 := "TestListAccessScopes_dynamicOriginTraits"
	scope1 := s.tester.createAccessScope(t, accessScopeName1, nilTraits)
	scope2 := s.tester.createAccessScope(t, accessScopeName2, imperativeOriginTraits)
	scope3 := s.tester.createAccessScope(t, accessScopeName3, declarativeOriginTraits)
	scope4 := s.tester.createAccessScope(t, accessScopeName4, orphanedDeclarativeOriginTraits)
	scope5 := s.tester.createAccessScope(t, accessScopeName5, dynamicOriginTraits)

	scopes, err := s.tester.service.ListSimpleAccessScopes(ctx, &v1.Empty{})
	s.NoError(err)
	s.Len(scopes.GetAccessScopes(), 4)

	protoassert.SliceContains(s.T(), scopes.GetAccessScopes(), scope1)
	protoassert.SliceContains(s.T(), scopes.GetAccessScopes(), scope2)
	protoassert.SliceContains(s.T(), scopes.GetAccessScopes(), scope3)
	protoassert.SliceContains(s.T(), scopes.GetAccessScopes(), scope4)
	// Roles with dynamic origin are filtered out.
	protoassert.SliceNotContains(s.T(), scopes.GetAccessScopes(), scope5)
}
