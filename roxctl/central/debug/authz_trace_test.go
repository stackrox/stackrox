package debug

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	mockEnv "github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()
)

func TestStreamAuthzTraces(t *testing.T) {
	svc := &fakeService{}
	conn, closeFunc, err := pkgGRPC.CreateTestGRPCStreamingService(
		context.Background(),
		t,
		func(registrar grpc.ServiceRegistrar) {
			v1.RegisterDebugServiceServer(registrar, svc)
		},
	)
	require.NoError(t, err)
	defer closeFunc()

	mockController := gomock.NewController(t)
	env := mockEnv.NewMockEnvironment(mockController)
	env.EXPECT().GRPCConnection().Times(1).Return(conn, nil)

	var buf bytes.Buffer
	streamErr := streamAuthzTraces(env, 10*time.Second, &buf)
	assert.NoError(t, streamErr)
	assert.JSONEq(t, expectedSerializedTrace, buf.String())

}

type fakeService struct{}

func (s *fakeService) GetLogLevel(_ context.Context, _ *v1.GetLogLevelRequest) (*v1.LogLevelResponse, error) {
	llr := &v1.LogLevelResponse{}
	llr.SetLevel("Invalid")
	return llr, nil
}

func (s *fakeService) SetLogLevel(_ context.Context, _ *v1.LogLevelRequest) (*protocompat.Empty, error) {
	return protocompat.ProtoEmpty(), nil
}

var (
	arrivedAtTime    = time.Date(2020, time.December, 24, 23, 59, 59, 999999999, time.UTC)
	processedAtTime  = time.Date(2020, time.December, 31, 23, 59, 59, 999999999, time.UTC)
	adminPermissions = map[string]storage.Access{
		"Access":                           storage.Access_READ_WRITE_ACCESS,
		"Administration":                   storage.Access_READ_WRITE_ACCESS,
		"Alert":                            storage.Access_READ_WRITE_ACCESS,
		"CVE":                              storage.Access_READ_WRITE_ACCESS,
		"Cluster":                          storage.Access_READ_WRITE_ACCESS,
		"Compliance":                       storage.Access_READ_WRITE_ACCESS,
		"Deployment":                       storage.Access_READ_WRITE_ACCESS,
		"DeploymentExtension":              storage.Access_READ_WRITE_ACCESS,
		"Detection":                        storage.Access_READ_WRITE_ACCESS,
		"Image":                            storage.Access_READ_WRITE_ACCESS,
		"Integration":                      storage.Access_READ_WRITE_ACCESS,
		"K8sRole":                          storage.Access_READ_WRITE_ACCESS,
		"K8sRoleBinding":                   storage.Access_READ_WRITE_ACCESS,
		"K8sSubject":                       storage.Access_READ_WRITE_ACCESS,
		"Namespace":                        storage.Access_READ_WRITE_ACCESS,
		"NetworkGraph":                     storage.Access_READ_WRITE_ACCESS,
		"NetworkPolicy":                    storage.Access_READ_WRITE_ACCESS,
		"Node":                             storage.Access_READ_WRITE_ACCESS,
		"Secret":                           storage.Access_READ_WRITE_ACCESS,
		"ServiceAccount":                   storage.Access_READ_WRITE_ACCESS,
		"VulnerabilityManagementApprovals": storage.Access_READ_WRITE_ACCESS,
		"VulnerabilityManagementRequests":  storage.Access_READ_WRITE_ACCESS,
		"WatchedImages":                    storage.Access_READ_WRITE_ACCESS,
		"WorkflowAdministration":           storage.Access_READ_WRITE_ACCESS,
	}
	trace = v1.AuthorizationTraceResponse_builder{
		ArrivedAt:   protocompat.ConvertTimeToTimestampOrNil(&arrivedAtTime),
		ProcessedAt: protocompat.ConvertTimeToTimestampOrNil(&processedAtTime),
		Request: v1.AuthorizationTraceResponse_Request_builder{
			Endpoint: "/api/graphql",
			Method:   http.MethodPost,
		}.Build(),
		Response: v1.AuthorizationTraceResponse_Response_builder{
			Status: http.StatusOK,
			Error:  "",
		}.Build(),
		User: v1.AuthorizationTraceResponse_User_builder{
			Username:              "admin",
			FriendlyName:          "admin",
			AggregatedPermissions: adminPermissions,
			Roles: []*v1.AuthorizationTraceResponse_User_Role{
				v1.AuthorizationTraceResponse_User_Role_builder{
					Name:            "admin",
					Permissions:     adminPermissions,
					AccessScopeName: "admin",
					AccessScope: storage.SimpleAccessScope_Rules_builder{
						IncludedClusters: []string{"*"},
					}.Build(),
				}.Build(),
			},
		}.Build(),
		Trace: v1.AuthorizationTraceResponse_Trace_builder{
			ScopeCheckerType: "built-in",
			BuiltIn: v1.AuthorizationTraceResponse_Trace_BuiltInAuthorizer_builder{
				ClustersTotalNum:      0,
				NamespacesTotalNum:    0,
				DeniedAuthzDecisions:  nil,
				AllowedAuthzDecisions: nil,
				EffectiveAccessScopes: map[string]string{"*": "*"},
			}.Build(),
		}.Build(),
	}.Build()

	serializedPermissions = `{
	"Access": "READ_WRITE_ACCESS",
	"Administration": "READ_WRITE_ACCESS",
	"Alert": "READ_WRITE_ACCESS",
	"CVE": "READ_WRITE_ACCESS",
	"Cluster": "READ_WRITE_ACCESS",
	"Compliance": "READ_WRITE_ACCESS",
	"Deployment": "READ_WRITE_ACCESS",
	"DeploymentExtension": "READ_WRITE_ACCESS",
	"Detection": "READ_WRITE_ACCESS",
	"Image": "READ_WRITE_ACCESS",
	"Integration": "READ_WRITE_ACCESS",
	"K8sRole": "READ_WRITE_ACCESS",
	"K8sRoleBinding": "READ_WRITE_ACCESS",
	"K8sSubject": "READ_WRITE_ACCESS",
	"Namespace": "READ_WRITE_ACCESS",
	"NetworkGraph": "READ_WRITE_ACCESS",
	"NetworkPolicy": "READ_WRITE_ACCESS",
	"Node": "READ_WRITE_ACCESS",
	"Secret": "READ_WRITE_ACCESS",
	"ServiceAccount": "READ_WRITE_ACCESS",
	"VulnerabilityManagementApprovals": "READ_WRITE_ACCESS",
	"VulnerabilityManagementRequests": "READ_WRITE_ACCESS",
	"WatchedImages": "READ_WRITE_ACCESS",
	"WorkflowAdministration": "READ_WRITE_ACCESS"
}`
	expectedSerializedTrace = `{
	"arrivedAt": "2020-12-24T23:59:59.999999999Z",
	"processedAt": "2020-12-31T23:59:59.999999999Z",
	"request": {
		"endpoint": "/api/graphql",
		"method": "POST"
	},
	"response": {
		"status": 200
	},
	"trace": {
		"builtIn": { "effectiveAccessScopes": {"*":"*"} },
		"scopeCheckerType": "built-in"
	},
	"user": {
		"aggregatedPermissions": ` + serializedPermissions + `,
		"friendlyName": "admin",
		"roles": [
			{
				"accessScope": {"includedClusters": ["*"]},
				"accessScopeName": "admin",
				"name": "admin",
				"permissions": ` + serializedPermissions + `
			}
		],
		"username": "admin"
	}
}`
)

func (s *fakeService) StreamAuthzTraces(_ *v1.Empty, stream v1.DebugService_StreamAuthzTracesServer) error {
	err := stream.Send(trace)
	if err != nil {
		if err != io.EOF {
			log.Warnf("Error during authz trace streaming: %s", err.Error())
		}
		return err
	}
	return nil
}

func (s *fakeService) ResetDBStats(_ context.Context, _ *v1.Empty) (*v1.Empty, error) {
	return &v1.Empty{}, nil
}
