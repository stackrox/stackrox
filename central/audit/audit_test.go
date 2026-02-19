package audit

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	tokenServiceV1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	identityMocks "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/grpc/authz/interceptor"
	notifierMocks "github.com/stackrox/rox/pkg/notifier/mocks"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
)

type AuditLogTestSuite struct {
	suite.Suite

	notifierMock *notifierMocks.MockProcessor
	identityMock *identityMocks.MockIdentity
}

func (suite *AuditLogTestSuite) SetupTest() {
	suite.notifierMock = notifierMocks.NewMockProcessor(gomock.NewController(suite.T()))
	suite.identityMock = identityMocks.NewMockIdentity(gomock.NewController(suite.T()))
}

func TestAuditLog(t *testing.T) {
	suite.Run(t, new(AuditLogTestSuite))
}

func (suite *AuditLogTestSuite) TestCalculateAuditStatus() {
	var wg sync.WaitGroup

	audit := New(suite.notifierMock)
	interceptorFunc := audit.UnaryServerInterceptor()
	serverInfo := &grpc.UnaryServerInfo{
		Server:     nil,
		FullMethod: "testMethod",
	}
	ctxWithNoAuth := context.Background()
	ctxAuthorised := interceptor.ContextWithAuthStatus(ctxWithNoAuth, nil)

	auditMessage := (*v1.Audit_Message)(nil)
	suite.notifierMock.EXPECT().HasEnabledAuditNotifiers().Times(4).Return(true)
	suite.notifierMock.EXPECT().ProcessAuditMessage(gomock.Any(), gomock.Any()).Times(4).Do(func(ctx context.Context, msg *v1.Audit_Message) {
		auditMessage = msg
		wg.Done()
	})

	// auth error --> AUTH_FAILED
	wg.Add(1)
	_, _ = interceptorFunc(ctxWithNoAuth, nil, serverInfo, handler(nil))
	wg.Wait()
	suite.Equal(v1.Audit_AUTH_FAILED, auditMessage.GetStatus())

	// request error --> REQUEST_FAILED
	wg.Add(1)
	err := errors.New("test error")
	_, _ = interceptorFunc(ctxAuthorised, nil, serverInfo, handler(err))
	wg.Wait()
	suite.Equal(v1.Audit_REQUEST_FAILED, auditMessage.GetStatus())
	suite.Equal("test error", auditMessage.GetStatusReason())

	// rejected by SAC --> AUTH_FAILED
	wg.Add(1)
	_, _ = interceptorFunc(ctxAuthorised, nil, serverInfo, handler(sac.ErrResourceAccessDenied))
	wg.Wait()
	suite.Equal(v1.Audit_AUTH_FAILED, auditMessage.GetStatus())

	// no error --> REQUEST_SUCCEEDED
	wg.Add(1)
	_, _ = interceptorFunc(ctxAuthorised, nil, serverInfo, handler(nil))
	wg.Wait()
	suite.Equal(v1.Audit_REQUEST_SUCCEEDED, auditMessage.GetStatus())
}

func (suite *AuditLogTestSuite) TestPermissionsRemoval() {
	userInfo := &storage.UserInfo{
		Username:     "sample-user",
		FriendlyName: "friendly-sample-user",
		Permissions: &storage.UserInfo_ResourceToAccess{
			ResourceToAccess: map[string]storage.Access{
				resources.Administration.String(): storage.Access_READ_ACCESS,
				resources.Integration.String():    storage.Access_READ_WRITE_ACCESS,
				resources.Access.String():         storage.Access_NO_ACCESS,
			},
		},
		Roles: []*storage.UserInfo_Role{
			{
				Name: "sample-role",
				ResourceToAccess: map[string]storage.Access{
					resources.Administration.String(): storage.Access_READ_ACCESS,
				},
			},
			{
				Name: "yet-another-sample-role",
				ResourceToAccess: map[string]storage.Access{
					resources.Integration.String(): storage.Access_READ_WRITE_ACCESS,
				},
			},
		},
	}
	suite.identityMock.EXPECT().Service().Return(nil).AnyTimes()
	suite.identityMock.EXPECT().User().Return(userInfo).AnyTimes()

	ctxWithMockIdentity := authn.ContextWithIdentity(context.Background(), suite.identityMock,
		suite.T())

	a := &audit{}
	withPermissions := a.newAuditMessage(ctxWithMockIdentity, "this is a test", "/v1./Test",
		interceptor.AuthStatus{Error: nil}, nil)

	protoassert.Equal(suite.T(), userInfo, withPermissions.GetUser())

	a = &audit{withoutPermissions: true}
	withoutPermissions := a.newAuditMessage(ctxWithMockIdentity, "this is a test", "/v1./Test",
		interceptor.AuthStatus{Error: nil}, nil)
	protoassert.NotEqual(suite.T(), userInfo, withoutPermissions.GetUser())
	suite.Empty(withoutPermissions.GetUser().GetPermissions())
	for _, userRole := range withoutPermissions.GetUser().GetRoles() {
		suite.Empty(userRole.GetResourceToAccess())
	}
}

func (suite *AuditLogTestSuite) TestServiceRequestsForInternalTokenEndpointAreAudited() {
	// Create mock service identity.
	serviceIdentity := &storage.ServiceIdentity{
		Id:   "test-sensor-12345",
		Type: storage.ServiceType_SENSOR_SERVICE,
	}
	suite.identityMock.EXPECT().Service().Return(serviceIdentity).AnyTimes()
	suite.identityMock.EXPECT().User().Return(nil).AnyTimes()

	ctxWithServiceIdentity := authn.ContextWithIdentity(context.Background(), suite.identityMock, suite.T())
	ctxWithAuth := interceptor.ContextWithAuthStatus(ctxWithServiceIdentity, nil)

	a := &audit{notifications: suite.notifierMock}

	// Test internal token generation endpoint - should be audited.
	msg := a.newAuditMessage(ctxWithAuth, "test-request",
		tokenServiceV1.TokenService_GenerateTokenForPermissionsAndScope_FullMethodName,
		interceptor.AuthStatus{Error: nil}, nil)

	suite.NotNil(msg, "Service request for internal token endpoint should be audited")
	suite.NotNil(msg.GetUser(), "Audit message should contain user info from service identity")
	suite.Equal("service:SENSOR_SERVICE:test-sensor-12345", msg.GetUser().GetUsername())
	suite.Equal("Service: SENSOR_SERVICE (ID: test-sensor-12345)", msg.GetUser().GetFriendlyName())
	suite.Equal(v1.Audit_API, msg.GetMethod(), "Service requests should have method=API")
	suite.Equal(tokenServiceV1.TokenService_GenerateTokenForPermissionsAndScope_FullMethodName,
		msg.GetRequest().GetEndpoint(), "Audit message should contain the correct endpoint")
	suite.Equal(v1.Audit_UPDATE, msg.GetInteraction(), "gRPC requests should have interaction=UPDATE")
}

func (suite *AuditLogTestSuite) TestServiceRequestsForOtherEndpointsAreNotAudited() {
	// Create mock service identity.
	serviceIdentity := &storage.ServiceIdentity{
		Id:   "test-sensor-12345",
		Type: storage.ServiceType_SENSOR_SERVICE,
	}
	suite.identityMock.EXPECT().Service().Return(serviceIdentity).AnyTimes()

	ctxWithServiceIdentity := authn.ContextWithIdentity(context.Background(), suite.identityMock, suite.T())
	ctxWithAuth := interceptor.ContextWithAuthStatus(ctxWithServiceIdentity, nil)

	a := &audit{notifications: suite.notifierMock}

	// Test non-token endpoint - should NOT be audited.
	msg := a.newAuditMessage(ctxWithAuth, "test-request",
		"/v1.SomeOtherService/SomeMethod",
		interceptor.AuthStatus{Error: nil}, nil)

	suite.Nil(msg, "Service request for non-token endpoint should not be audited")
}

func (suite *AuditLogTestSuite) TestServiceRequestsForInternalTokenEndpointWithAuthErrorAreAudited() {
	// Create mock service identity.
	serviceIdentity := &storage.ServiceIdentity{
		Id:   "test-sensor-12345",
		Type: storage.ServiceType_SENSOR_SERVICE,
	}
	suite.identityMock.EXPECT().Service().Return(serviceIdentity).AnyTimes()
	suite.identityMock.EXPECT().User().Return(nil).AnyTimes()

	ctxWithServiceIdentity := authn.ContextWithIdentity(context.Background(), suite.identityMock, suite.T())
	ctxWithAuth := interceptor.ContextWithAuthStatus(ctxWithServiceIdentity, nil)

	a := &audit{notifications: suite.notifierMock}

	// Simulate an auth failure for the internal token endpoint.
	authErr := errors.New("test auth error")
	msg := a.newAuditMessage(ctxWithAuth, "test-request",
		tokenServiceV1.TokenService_GenerateTokenForPermissionsAndScope_FullMethodName,
		interceptor.AuthStatus{Error: authErr}, nil)

	suite.NotNil(msg, "Service request for internal token endpoint with auth error should still be audited")
	suite.NotNil(msg.GetUser(), "Audit message should contain user info from service identity")
	suite.Equal("service:SENSOR_SERVICE:test-sensor-12345", msg.GetUser().GetUsername())
	suite.Equal("Service: SENSOR_SERVICE (ID: test-sensor-12345)", msg.GetUser().GetFriendlyName())
	suite.Equal(v1.Audit_API, msg.GetMethod(), "Service requests should have method=API")
	suite.Equal(v1.Audit_AUTH_FAILED, msg.GetStatus(), "Auth error should result in AUTH_FAILED status")
	suite.Contains(msg.GetStatusReason(), authErr.Error(), "Audit message should contain auth error details")
}

func (suite *AuditLogTestSuite) TestUserRequestsContinueToBeAudited() {
	userInfo := &storage.UserInfo{
		Username:     "test-user",
		FriendlyName: "Test User",
	}
	suite.identityMock.EXPECT().Service().Return(nil).AnyTimes()
	suite.identityMock.EXPECT().User().Return(userInfo).AnyTimes()

	ctxWithUserIdentity := authn.ContextWithIdentity(context.Background(), suite.identityMock, suite.T())
	ctxWithAuth := interceptor.ContextWithAuthStatus(ctxWithUserIdentity, nil)

	a := &audit{notifications: suite.notifierMock}

	// Test any endpoint with user identity - should be audited.
	msg := a.newAuditMessage(ctxWithAuth, "test-request",
		"/v1.SomeService/SomeMethod",
		interceptor.AuthStatus{Error: nil}, nil)

	suite.NotNil(msg, "User request should be audited")
	suite.NotNil(msg.GetUser(), "Audit message should contain user info")
	suite.Equal("test-user", msg.GetUser().GetUsername())
	suite.Equal(v1.Audit_CLI, msg.GetMethod(), "User gRPC requests should have method=CLI")
}

func handler(err error) func(ctx context.Context, req interface{}) (interface{}, error) {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, err
	}
}
