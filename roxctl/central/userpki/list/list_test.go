package list

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders/userpki"
	"github.com/stackrox/rox/pkg/errox"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	envMocks "github.com/stackrox/rox/roxctl/common/environment/mocks"
	ioMocks "github.com/stackrox/rox/roxctl/common/io/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
)

func TestListProvidersOptJSON(t *testing.T) {

	fakeService := &fakeAccessService{tb: t}
	conn, closeFunc, err := pkgGRPC.CreateTestGRPCStreamingService(
		context.Background(),
		t,
		func(registrar grpc.ServiceRegistrar) {
			v1.RegisterAuthProviderServiceServer(registrar, fakeService)
			v1.RegisterGroupServiceServer(registrar, fakeService)
		},
	)
	require.NoError(t, err)
	defer closeFunc()

	mockCtrl := gomock.NewController(t)
	var buf bytes.Buffer

	mockIO := ioMocks.NewMockIO(mockCtrl)
	mockIO.EXPECT().Out().Times(1).Return(&buf)

	mockEnv := envMocks.NewMockEnvironment(mockCtrl)
	mockEnv.EXPECT().GRPCConnection(gomock.Any()).Times(1).Return(conn, nil)
	mockEnv.EXPECT().InputOutput().Times(1).Return(mockIO)

	mockLogger := &fakeLogger{}
	mockEnv.EXPECT().Logger().AnyTimes().Return(mockLogger)

	cmd := &centralUserPkiListCommand{
		env:          mockEnv,
		json:         true,
		timeout:      10 * time.Second,
		retryTimeout: 10 * time.Second,
	}
	err = cmd.listProviders()
	assert.NoError(t, err)
	assert.JSONEq(t, expectedSerializedAuthProviders, buf.String())
	assert.Contains(t, buf.String(), "\n  \"authProviders\"")
	assert.Contains(t, buf.String(), "\n    {")
	assert.Contains(t, buf.String(), "\n      \"id\"")
	assert.Contains(t, buf.String(), "\n      \"name\"")
	assert.Contains(t, buf.String(), "\n      \"type\"")

	fmt.Println(mockLogger.buf.String())
}

var (
	expectedSerializedAuthProviders = `{
    "authProviders": [
        {
            "id": "41757468-5072-4011-cccc-111111111111",
            "name": "UserPKI provider 1",
            "type": "userpki"
        },
        {
            "id": "41757468-5072-4011-cccc-222222222222",
            "name": "UserPKI provider 2",
            "type": "userpki"
        }
    ]
}`
)

type fakeLogger struct {
	buf bytes.Buffer
}

func (l *fakeLogger) ErrfLn(format string, args ...interface{}) {
	l.println("ERROR: ", format, args...)
}

func (l *fakeLogger) WarnfLn(format string, args ...interface{}) {
	l.println("WARN: ", format, args...)
}

func (l *fakeLogger) InfofLn(format string, args ...interface{}) {
	l.println("INFO: ", format, args...)
}

func (l *fakeLogger) PrintfLn(format string, args ...interface{}) {
	l.println("", format, args...)
}

func (l *fakeLogger) println(prefix string, format string, args ...interface{}) {
	l.buf.WriteString(fmt.Sprintf(prefix+format, args...))
}

type fakeAccessService struct {
	tb testing.TB
}

const (
	authProviderId1 = "41757468-5072-4011-cccc-111111111111"
	authProviderId2 = "41757468-5072-4011-cccc-222222222222"
)

func (s *fakeAccessService) GetAuthProviders(_ context.Context, _ *v1.GetAuthProvidersRequest) (*v1.GetAuthProvidersResponse, error) {
	ap := &storage.AuthProvider{}
	ap.SetId(authProviderId1)
	ap.SetName("UserPKI provider 1")
	ap.SetType(userpki.TypeName)
	ap2 := &storage.AuthProvider{}
	ap2.SetId(authProviderId2)
	ap2.SetName("UserPKI provider 2")
	ap2.SetType(userpki.TypeName)
	gapr := &v1.GetAuthProvidersResponse{}
	gapr.SetAuthProviders([]*storage.AuthProvider{
		ap,
		ap2,
	})
	return gapr, nil
}

func (s *fakeAccessService) GetGroups(_ context.Context, _ *v1.GetGroupsRequest) (*v1.GetGroupsResponse, error) {
	return v1.GetGroupsResponse_builder{
		Groups: []*storage.Group{
			storage.Group_builder{
				Props: storage.GroupProperties_builder{
					AuthProviderId: authProviderId1,
					Key:            "",
				}.Build(),
				RoleName: "Continuous Integration",
			}.Build(),
			storage.Group_builder{
				Props: storage.GroupProperties_builder{
					AuthProviderId: authProviderId1,
					Key:            "email",
					Value:          "no-reply@stackrox.io",
				}.Build(),
				RoleName: "Admin",
			}.Build(),
			storage.Group_builder{
				Props: storage.GroupProperties_builder{
					AuthProviderId: authProviderId2,
					Key:            "",
				}.Build(),
				RoleName: "Analyst",
			}.Build(),
			storage.Group_builder{
				Props: storage.GroupProperties_builder{
					AuthProviderId: "authProviderId3",
					Key:            "",
				}.Build(),
				RoleName: "Admin",
			}.Build(),
		},
	}.Build(), nil
}

func (s *fakeAccessService) GetAuthProvider(_ context.Context, _ *v1.GetAuthProviderRequest) (*storage.AuthProvider, error) {
	return nil, errox.NotImplemented
}

func (s *fakeAccessService) GetLoginAuthProviders(_ context.Context, _ *v1.Empty) (*v1.GetLoginAuthProvidersResponse, error) {
	return nil, errox.NotImplemented
}

func (s *fakeAccessService) ListAvailableProviderTypes(_ context.Context, _ *v1.Empty) (*v1.AvailableProviderTypesResponse, error) {
	return nil, errox.NotImplemented
}

func (s *fakeAccessService) PostAuthProvider(_ context.Context, _ *v1.PostAuthProviderRequest) (*storage.AuthProvider, error) {
	return nil, errox.NotImplemented
}

func (s *fakeAccessService) PutAuthProvider(_ context.Context, _ *storage.AuthProvider) (*storage.AuthProvider, error) {
	return nil, errox.NotImplemented
}

func (s *fakeAccessService) UpdateAuthProvider(_ context.Context, _ *v1.UpdateAuthProviderRequest) (*storage.AuthProvider, error) {
	return nil, errox.NotImplemented
}

func (s *fakeAccessService) DeleteAuthProvider(_ context.Context, _ *v1.DeleteByIDWithForce) (*v1.Empty, error) {
	return nil, errox.NotImplemented
}

func (s *fakeAccessService) ExchangeToken(_ context.Context, _ *v1.ExchangeTokenRequest) (*v1.ExchangeTokenResponse, error) {
	return nil, errox.NotImplemented
}

func (s *fakeAccessService) GetGroup(_ context.Context, _ *storage.GroupProperties) (*storage.Group, error) {
	return nil, errox.NotImplemented
}

func (s *fakeAccessService) BatchUpdate(_ context.Context, _ *v1.GroupBatchUpdateRequest) (*v1.Empty, error) {
	return nil, errox.NotImplemented
}

func (s *fakeAccessService) CreateGroup(_ context.Context, _ *storage.Group) (*v1.Empty, error) {
	return nil, errox.NotImplemented
}

func (s *fakeAccessService) UpdateGroup(_ context.Context, _ *v1.UpdateGroupRequest) (*v1.Empty, error) {
	return nil, errox.NotImplemented
}

func (s *fakeAccessService) DeleteGroup(_ context.Context, _ *v1.DeleteGroupRequest) (*v1.Empty, error) {
	return nil, errox.NotImplemented
}
