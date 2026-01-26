//go:build sql_integration

package service

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

func TestServiceImplWithDB_Roles(t *testing.T) {
	suite.Run(t, new(serviceImplRoleTestSuite))
}

type serviceImplRoleTestSuite struct {
	suite.Suite

	tester *serviceImplTester
}

func (s *serviceImplRoleTestSuite) SetupSuite() {
	s.tester = &serviceImplTester{}
	s.tester.Setup(s.T())
}

func (s *serviceImplRoleTestSuite) SetupTest() {
	s.Require().NotNil(s.tester)
	s.tester.SetupTest(s.T())
}

func (s *serviceImplRoleTestSuite) TearDownTest() {
	s.Require().NotNil(s.tester)
	s.tester.TearDownTest(s.T())
}

func (s *serviceImplRoleTestSuite) TestCreateRoleValidAccessScopeID() {
	t := s.T()
	ctx := sac.WithAllAccess(t.Context())
	roleName := "TestCreateRoleValidAccessScopeID"

	ps := s.tester.createPermissionSet(t, roleName, nilTraits)
	scope := s.tester.createAccessScope(t, roleName, nilTraits)

	role := getValidRole(roleName)
	role.PermissionSetId = ps.GetId()
	role.AccessScopeId = scope.GetId()
	createRoleRequest := &v1.CreateRoleRequest{
		Name: roleName,
		Role: role,
	}
	_, err := s.tester.service.CreateRole(ctx, createRoleRequest)
	s.NoError(err)
	s.tester.storedRoleNames = append(s.tester.storedRoleNames, role.GetName())
}

func (s *serviceImplRoleTestSuite) TestCreateRoleEmptyAccessScopeID() {
	t := s.T()
	ctx := sac.WithAllAccess(t.Context())
	roleName := "TestCreateRoleEmptyAccessScopeID"

	ps := s.tester.createPermissionSet(t, roleName, nilTraits)

	role := getValidRole(roleName)
	role.PermissionSetId = ps.GetId()
	role.AccessScopeId = ""
	createRoleRequest := &v1.CreateRoleRequest{
		Name: roleName,
		Role: role,
	}
	_, err := s.tester.service.CreateRole(ctx, createRoleRequest)
	s.ErrorContains(err, "role access_scope_id field must be set")
}

func (s *serviceImplRoleTestSuite) TestUpdateExistingRoleValidAccessScopeID() {
	t := s.T()
	ctx := sac.WithAllAccess(t.Context())
	roleName := "TestUpdateExistingRoleValidAccessScopeID"
	role := s.tester.createRole(t, roleName, nilTraits)
	newScope := s.tester.createAccessScope(t, "new scope", nilTraits)
	role.AccessScopeId = newScope.GetId()
	_, err := s.tester.service.UpdateRole(ctx, role)
	s.NoError(err)
}

func (s *serviceImplRoleTestSuite) TestUpdateExistingRoleEmptyAccessScopeID() {
	t := s.T()
	ctx := sac.WithAllAccess(t.Context())
	roleName := "TestUpdateExistingRoleEmptyAccessScopeID"
	role := s.tester.createRole(t, roleName, nilTraits)
	role.AccessScopeId = ""
	_, err := s.tester.service.UpdateRole(ctx, role)
	s.ErrorContains(err, "role access_scope_id field must be set")
}

func (s *serviceImplRoleTestSuite) TestUpdateMissingRoleValidAccessScopeID() {
	t := s.T()
	ctx := sac.WithAllAccess(t.Context())
	roleName := "TestUpdateMissingRoleValidAccessScopeID"
	ps := s.tester.createPermissionSet(t, roleName, nilTraits)
	scope := s.tester.createAccessScope(t, roleName, nilTraits)
	role := getValidRole(roleName)
	role.PermissionSetId = ps.GetId()
	role.AccessScopeId = scope.GetId()
	_, err := s.tester.service.UpdateRole(ctx, role)
	s.ErrorIs(err, errox.NotFound)
}

func (s *serviceImplRoleTestSuite) TestUpdateMissingRoleEmptyAccessScopeID() {
	t := s.T()
	ctx := sac.WithAllAccess(t.Context())
	roleName := "TestUpdateMissingRoleEmptyAccessScopeID"
	ps := s.tester.createPermissionSet(t, roleName, nilTraits)
	role := getValidRole(roleName)
	role.PermissionSetId = ps.GetId()
	role.AccessScopeId = ""
	_, err := s.tester.service.UpdateRole(ctx, role)
	s.ErrorContains(err, "role access_scope_id field must be set")
}

func (s *serviceImplRoleTestSuite) TestGetRoles() {
	t := s.T()
	ctx := sac.WithAllAccess(t.Context())

	roleName1 := "TestGetRoles_noTraits"
	roleName2 := "TestGetRoles_imperativeOriginTraits"
	roleName3 := "TestGetRoles_declarativeOriginTraits"
	roleName4 := "TestGetRoles_orphanedDeclarativeOriginTraits"
	roleName5 := "TestGetRoles_dynamicOriginTraits"
	role1 := s.tester.createRole(t, roleName1, nilTraits)
	role2 := s.tester.createRole(t, roleName2, imperativeOriginTraits)
	role3 := s.tester.createRole(t, roleName3, declarativeOriginTraits)
	role4 := s.tester.createRole(t, roleName4, orphanedDeclarativeOriginTraits)
	role5 := s.tester.createRole(t, roleName5, dynamicOriginTraits)

	roles, err := s.tester.service.GetRoles(ctx, &v1.Empty{})
	s.NoError(err)
	s.Len(roles.GetRoles(), 4)

	protoassert.SliceContains(s.T(), roles.GetRoles(), role1)
	protoassert.SliceContains(s.T(), roles.GetRoles(), role2)
	protoassert.SliceContains(s.T(), roles.GetRoles(), role3)
	protoassert.SliceContains(s.T(), roles.GetRoles(), role4)
	// Roles with dynamic origin are filtered out.
	protoassert.SliceNotContains(s.T(), roles.GetRoles(), role5)
}
