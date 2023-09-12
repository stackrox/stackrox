package service

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

var (
	targetEndPointNames = []string{
		"/central.DevelopmentService/ReplicateImage",
		"/central.DevelopmentService/URLHasValidCert",
		"/central.DevelopmentService/RandomData",
		"/central.DevelopmentService/EnvVars",
		"/central.DevelopmentService/ReconciliationStatsByCluster",
	}
)

func TestDevelopmentServiceAccessControl(t *testing.T) {
	suite.Run(t, new(developmentServiceAccessControlTestSuite))
}

type developmentServiceAccessControlTestSuite struct {
	suite.Suite

	svc *serviceImpl

	authProvider authproviders.Provider

	withAdminRoleCtx context.Context
	withNoneRoleCtx  context.Context
	withNoAccessCtx  context.Context
	withNoRoleCtx    context.Context
	anonymousCtx     context.Context
}

func (s *developmentServiceAccessControlTestSuite) SetupSuite() {
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

	expectedAuthorizerError    error
	expectedRandomServiceError error
}

func (s *developmentServiceAccessControlTestSuite) getTestCases() []testCase {
	return []testCase{
		{
			name: accesscontrol.Admin,
			ctx:  s.withAdminRoleCtx,

			expectedRandomServiceError: nil,
			expectedAuthorizerError:    nil,
		},
		{
			name: accesscontrol.None,
			ctx:  s.withNoneRoleCtx,

			expectedRandomServiceError: nil,
			expectedAuthorizerError:    errox.NotAuthorized,
		},
		{
			name: "No Access",
			ctx:  s.withNoAccessCtx,

			expectedRandomServiceError: nil,
			expectedAuthorizerError:    errox.NotAuthorized,
		},
		{
			name: "No Role",
			ctx:  s.withNoRoleCtx,

			expectedRandomServiceError: nil,
			expectedAuthorizerError:    errox.NoCredentials,
		},
		{
			name: "Anonymous",
			ctx:  s.anonymousCtx,

			expectedRandomServiceError: nil,
			expectedAuthorizerError:    errox.NoCredentials,
		},
	}
}

func (s *developmentServiceAccessControlTestSuite) TestDevelopmentServiceAuthorizer() {
	for _, endPoint := range targetEndPointNames {
		s.Run(endPoint, func() {
			for _, c := range s.getTestCases() {
				s.Run(c.name, func() {
					ctx, err := s.svc.AuthFuncOverride(c.ctx, endPoint)
					s.ErrorIs(err, c.expectedAuthorizerError)
					s.Equal(c.ctx, ctx)
				})
			}
		})
	}
}

func (s *developmentServiceAccessControlTestSuite) TestDevelopmentServiceRandomBytes() {
	const dataSize = 16
	request := &central.RandomDataRequest{
		Size_: dataSize,
	}
	for _, c := range s.getTestCases() {
		s.Run(c.name, func() {
			rsp, err := s.svc.RandomData(c.ctx, request)
			s.ErrorIs(err, c.expectedRandomServiceError)
			s.NotNil(rsp)
			if rsp != nil {
				s.Len(rsp.GetData(), dataSize)
			}
		})
	}
}
