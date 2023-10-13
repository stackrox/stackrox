package service

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

const (
	targetEndPointName = "/v1.AuthService/GetAuthStatus"
)

func TestAuthServiceAccessControl(t *testing.T) {
	suite.Run(t, new(authServiceAccessControlTestSuite))
}

type authServiceAccessControlTestSuite struct {
	suite.Suite

	svc Service

	authProvider authproviders.Provider

	withAdminRoleCtx context.Context
	withNoneRoleCtx  context.Context
	withNoAccessCtx  context.Context
	withNoRoleCtx    context.Context
	anonymousCtx     context.Context
}

func (s *authServiceAccessControlTestSuite) SetupSuite() {
	s.svc = &serviceImpl{}

	authProvider, err := authproviders.NewProvider(
		authproviders.WithEnabled(true),
		authproviders.WithID(uuid.NewDummy().String()),
		authproviders.WithName("Test Auth Provider"),
	)
	s.Require().NoError(err)
	s.authProvider = authProvider
	s.withAdminRoleCtx = basic.ContextWithAdminIdentity(s.T(), s.authProvider)
	s.withNoneRoleCtx = basic.ContextWithNoneIdentity(s.T(), s.authProvider)
	s.withNoAccessCtx = basic.ContextWithNoAccessIdentity(s.T(), s.authProvider)
	s.withNoRoleCtx = basic.ContextWithNoRoleIdentity(s.T(), s.authProvider)
	s.anonymousCtx = context.Background()
}

type testCase struct {
	name string
	ctx  context.Context

	expectedAuthorizerError error
	expectedServiceError    error
}

func (s *authServiceAccessControlTestSuite) getTestCases() []testCase {
	return []testCase{
		{
			name: accesscontrol.Admin,
			ctx:  s.withAdminRoleCtx,

			expectedServiceError:    nil,
			expectedAuthorizerError: nil,
		},
		{
			name: accesscontrol.None,
			ctx:  s.withNoneRoleCtx,

			expectedServiceError:    nil,
			expectedAuthorizerError: nil,
		},
		{
			name: "No Access",
			ctx:  s.withNoAccessCtx,

			expectedServiceError:    nil,
			expectedAuthorizerError: nil,
		},
		{
			name: "No Role",
			ctx:  s.withNoRoleCtx,

			expectedServiceError:    nil,
			expectedAuthorizerError: nil,
		},
		{
			name: "Anonymous",
			ctx:  s.anonymousCtx,

			expectedServiceError:    errox.NoCredentials,
			expectedAuthorizerError: nil,
		},
	}
}

func (s *authServiceAccessControlTestSuite) TestAuthServiceAuthorizer() {
	for _, c := range s.getTestCases() {
		s.Run(c.name, func() {
			ctx, err := s.svc.AuthFuncOverride(c.ctx, targetEndPointName)
			s.ErrorIs(err, c.expectedAuthorizerError)
			s.Equal(c.ctx, ctx)
		})
	}
}

func (s *authServiceAccessControlTestSuite) TestAuthServiceResponse() {
	emptyQuery := &v1.Empty{}
	for _, c := range s.getTestCases() {
		s.Run(c.name, func() {
			rsp, err := s.svc.GetAuthStatus(c.ctx, emptyQuery)
			s.ErrorIs(err, c.expectedServiceError)
			if c.expectedServiceError == nil {
				s.NotNil(rsp)
				s.Equal(c.name, rsp.GetUserInfo().GetUsername())
				s.Equal(uuid.NewDummy().String(), rsp.GetAuthProvider().GetId())
			} else {
				s.Nil(rsp)
			}
		})
	}
}
