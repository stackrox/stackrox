package tests

import (
	"context"
	"fmt"
	"testing"

	apiTokenService "github.com/stackrox/rox/central/apitoken/service"
	"github.com/stackrox/rox/central/role/service"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

const UnrestrictedScopeId = "ffffffff-ffff-fff4-f5ff-ffffffffffff"

type DiagnosticBundleTestSuite struct {
	suite.Suite
	roleService     service.Service
	ctx             context.Context
	apiTokenService apiTokenService.Service

	debugLogsReaderToken string
	noAccessToken        string
}

func (s *DiagnosticBundleTestSuite) TestDiagnosticBundle(t *testing.T) {
	s.apiTokenService = apiTokenService.Singleton()
	administratorReaderRoleName := uuid.NewV4().String()

	// Create reader role
	s.createRoleWithScopeAndPermissionSet(administratorReaderRoleName, UnrestrictedScopeId, map[string]storage.Access{
		"Administrator": storage.Access_READ_ACCESS,
		"Cluster":       storage.Access_READ_ACCESS,
	})
	debugLogsReaderToken, err := s.apiTokenService.GenerateToken(s.ctx, &v1.GenerateTokenRequest{
		Name:  uuid.NewV4().String(),
		Roles: []string{administratorReaderRoleName},
	})
	s.Require().NoError(err)
	s.debugLogsReaderToken = debugLogsReaderToken.GetToken()

	// Create no access role
	resourcesToAccess := map[string]storage.Access{
		"Administrator": storage.Access_NO_ACCESS,
		"Cluster":       storage.Access_NO_ACCESS,
	}
	//TODO: Add Run ID
	noAccessRole := s.createRoleWithScopeAndPermissionSet("No Access Test Role - ${RUN_ID}", UnrestrictedScopeId, resourcesToAccess)
	noAccessToken, err := s.apiTokenService.GenerateToken(s.ctx, &v1.GenerateTokenRequest{
		Name:  uuid.NewV4().String(),
		Roles: []string{noAccessRole.Name},
	})
	s.Require().NoError(err)
	s.noAccessToken = noAccessToken.GetToken()
}

func (s *DiagnosticBundleTestSuite) createRoleWithScopeAndPermissionSet(name string, accessScopeId string, resource map[string]storage.Access) storage.Role {
	permissionSet, _ := s.createPermissionSet(fmt.Sprintf("Test Automation Permission Set %s for %s", uuid.NewV4().String(), name), map[string]storage.Access{})

	role := storage.Role{
		Name:            name,
		AccessScopeId:   accessScopeId,
		PermissionSetId: permissionSet.GetId(),
	}
	_, err := s.roleService.CreateRole(s.ctx, &v1.CreateRoleRequest{
		Name: role.Name,
		Role: &role,
	})
	if err != nil {
		// TODO: require no error
	}
	return role
}

func (s *DiagnosticBundleTestSuite) createPermissionSet(name string, resources map[string]storage.Access) (*storage.PermissionSet, error) {
	return s.roleService.PostPermissionSet(s.ctx, &storage.PermissionSet{
		Name:             "",
		ResourceToAccess: map[string]storage.Access{},
	})
}
