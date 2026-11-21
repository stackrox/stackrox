package service

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestGetMyPermissions(t *testing.T) {
	suite.Run(t, new(roleServiceGetMyPermissionsTestSuite))
}

const (
	getMyPermissionsServiceName = "/v1.RoleService/GetMyPermissions"
)

type roleServiceGetMyPermissionsTestSuite struct {
	suite.Suite

	svc *serviceImpl

	withAdminRoleCtx context.Context
	withNoneRoleCtx  context.Context
	withNoAccessCtx  context.Context
	withNoRoleCtx    context.Context
	anonymousCtx     context.Context
}

func (s *roleServiceGetMyPermissionsTestSuite) SetupTest() {
	s.svc = &serviceImpl{}

	authProvider, err := authproviders.NewProvider(
		authproviders.WithEnabled(true),
		authproviders.WithID(uuid.NewDummy().String()),
		authproviders.WithName("Test Auth Provider"),
	)
	s.Require().NoError(err)
	s.withAdminRoleCtx = basic.ContextWithAdminIdentity(s.T(), authProvider)
	s.withNoneRoleCtx = basic.ContextWithNoneIdentity(s.T(), authProvider)
	s.withNoAccessCtx = basic.ContextWithNoAccessIdentity(s.T(), authProvider)
	s.withNoRoleCtx = basic.ContextWithNoRoleIdentity(s.T(), authProvider)
	s.anonymousCtx = context.Background()
}

type getMyPermissionsTestCase struct {
	name string
	ctx  context.Context

	expectedPermissionCount int
	expectedAuthorizerError error
	expectedServiceError    error
}

func (s *roleServiceGetMyPermissionsTestSuite) getTestCases() []getMyPermissionsTestCase {
	return []getMyPermissionsTestCase{
		{
			name: accesscontrol.Admin,
			ctx:  s.withAdminRoleCtx,

			expectedPermissionCount: len(resources.ListAll()),
			expectedServiceError:    nil,
			expectedAuthorizerError: nil,
		},
		{
			name: accesscontrol.None,
			ctx:  s.withNoneRoleCtx,

			expectedPermissionCount: 0,
			expectedServiceError:    nil,
			expectedAuthorizerError: errox.NoCredentials,
		},
		{
			name: "No Access",
			ctx:  s.withNoAccessCtx,

			expectedPermissionCount: len(resources.ListAll()),
			expectedServiceError:    nil,
			expectedAuthorizerError: nil,
		},
		{
			name: "No Role",
			ctx:  s.withNoRoleCtx,

			expectedPermissionCount: 0,
			expectedServiceError:    nil,
			expectedAuthorizerError: errox.NoCredentials,
		},
		{
			name: "Anonymous",
			ctx:  s.anonymousCtx,

			expectedPermissionCount: 0,
			expectedServiceError:    errox.NoCredentials,
			expectedAuthorizerError: errox.NoCredentials,
		},
	}
}

func (s *roleServiceGetMyPermissionsTestSuite) TestAuthorizer() {
	for _, c := range s.getTestCases() {
		s.Run(c.name, func() {
			ctx, err := s.svc.AuthFuncOverride(c.ctx, getMyPermissionsServiceName)
			s.ErrorIs(err, c.expectedAuthorizerError)
			s.Equal(c.ctx, ctx)
		})
	}
}

func (s *roleServiceGetMyPermissionsTestSuite) TestGetMyPermissions() {
	emptyRequest := &v1.Empty{}
	for _, c := range s.getTestCases() {
		s.Run(c.name, func() {
			rsp, err := s.svc.GetMyPermissions(c.ctx, emptyRequest)
			s.ErrorIs(err, c.expectedServiceError)
			if c.expectedServiceError == nil {
				s.NotNil(rsp)
				if rsp != nil {
					s.Len(rsp.GetResourceToAccess(), c.expectedPermissionCount)
				}
			} else {
				s.Nil(rsp)
			}
		})
	}
}
