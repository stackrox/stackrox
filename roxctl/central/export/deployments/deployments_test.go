package deployments

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
)

func TestDeployments(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	env := mocks.NewMockEnvironment(mockCtrl)

	fakeService := &fakeDeploymentService{}
	conn, closeFunc, err := pkgGRPC.CreateTestGRPCStreamingService(
		context.Background(),
		t,
		func(registrar grpc.ServiceRegistrar) {
			v1.RegisterDeploymentServiceServer(registrar, fakeService)
		},
	)
	require.NoError(t, err)
	defer closeFunc()
	envIO := newFakeIO()
	env.EXPECT().GRPCConnection().Times(1).Return(conn, nil)
	env.EXPECT().InputOutput().Times(1).Return(envIO)

	fakeCmd := &cobra.Command{}
	flags.AddTimeoutWithDefault(fakeCmd, 10*time.Second)

	cmd := Command(env)
	err = cmd.RunE(fakeCmd, []string{})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"deployment":`+expectedDeploymentJSON+`}`, envIO.out.String())
}

type fakeIO struct {
	in     *bytes.Buffer
	out    *bytes.Buffer
	errOut *bytes.Buffer
}

func newFakeIO() *fakeIO {
	return &fakeIO{
		in:     bytes.NewBuffer([]byte{}),
		out:    &bytes.Buffer{},
		errOut: &bytes.Buffer{},
	}
}

func (f *fakeIO) In() io.Reader {
	return f.in
}

func (f *fakeIO) Out() io.Writer {
	return f.out
}

func (f *fakeIO) ErrOut() io.Writer {
	return f.errOut
}

type fakeDeploymentService struct{}

func (s *fakeDeploymentService) GetDeployment(_ context.Context, _ *v1.ResourceByID) (*storage.Deployment, error) {
	return nil, errox.NotImplemented
}

func (s *fakeDeploymentService) GetDeploymentWithRisk(_ context.Context, _ *v1.ResourceByID) (*v1.GetDeploymentWithRiskResponse, error) {
	return nil, errox.NotImplemented
}

func (s *fakeDeploymentService) CountDeployments(_ context.Context, _ *v1.RawQuery) (*v1.CountDeploymentsResponse, error) {
	return nil, errox.NotImplemented
}

func (s *fakeDeploymentService) ListDeployments(_ context.Context, _ *v1.RawQuery) (*v1.ListDeploymentsResponse, error) {
	return nil, errox.NotImplemented
}

func (s *fakeDeploymentService) ListDeploymentsWithProcessInfo(_ context.Context, _ *v1.RawQuery) (*v1.ListDeploymentsWithProcessInfoResponse, error) {
	return nil, errox.NotImplemented
}

func (s *fakeDeploymentService) GetLabels(_ context.Context, _ *v1.Empty) (*v1.DeploymentLabelsResponse, error) {
	return nil, errox.NotImplemented
}

var (
	createdDate    = time.Date(2020, time.December, 24, 23, 59, 59, 999999999, time.UTC)
	testDeployment = &storage.Deployment{
		Id:                    fixtureconsts.Deployment1,
		Name:                  "Test Deployment",
		Hash:                  uint64(12345678901),
		Type:                  "Deployment",
		Namespace:             "Test Namespace",
		NamespaceId:           fixtureconsts.Namespace1,
		OrchestratorComponent: false,
		Replicas:              1,
		Labels:                map[string]string{"app": "Test Application"},
		PodLabels:             map[string]string{"app": "Test Application"},
		LabelSelector: &storage.LabelSelector{
			MatchLabels:  map[string]string{"app": "Test Application"},
			Requirements: []*storage.LabelSelector_Requirement{},
		},
		Created:     protocompat.ConvertTimeToTimestampOrNil(&createdDate),
		ClusterId:   fixtureconsts.Cluster1,
		ClusterName: "Test Cluster",
		Containers: []*storage.Container{
			{
				Id:     "ccaaaaaa-cccc-4011-0000-111111111111:testapplication",
				Config: &storage.ContainerConfig{},
				Image: &storage.ContainerImage{
					Id: "sha256:1234567890123456789012345678901234567890123456789012345678901234",
					Name: &storage.ImageName{
						Registry: "quay.io",
						Remote:   "test/application",
						Tag:      "demo",
						FullName: "quay.io/test/application:demo",
					},
				},
				SecurityContext: &storage.SecurityContext{},
				Volumes:         []*storage.Volume{},
				Ports: []*storage.PortConfig{
					{
						ContainerPort: 80,
						Protocol:      "TCP",
						Exposure:      storage.PortConfig_INTERNAL,
						ExposureInfos: []*storage.PortConfig_ExposureInfo{
							{
								Level:            storage.PortConfig_INTERNAL,
								ServiceName:      "test-application-service",
								ServiceId:        "faeeeeee-aaaa-4011-0000-111111111111",
								ServiceClusterIp: "127.0.0.1",
								ServicePort:      80,
							},
						},
					},
				},
				Secrets:   []*storage.EmbeddedSecret{},
				Resources: &storage.Resources{},
				Name:      "Test Application",
				LivenessProbe: &storage.LivenessProbe{
					Defined: false,
				},
				ReadinessProbe: &storage.ReadinessProbe{
					Defined: false,
				},
			},
		},
		Priority:                      42,
		ServiceAccount:                "default",
		ServiceAccountPermissionLevel: storage.PermissionLevel_NONE,
		AutomountServiceAccountToken:  true,
		Ports: []*storage.PortConfig{
			{
				ContainerPort: 80,
				Protocol:      "TCP",
				Exposure:      storage.PortConfig_INTERNAL,
				ExposureInfos: []*storage.PortConfig_ExposureInfo{
					{
						Level:            storage.PortConfig_INTERNAL,
						ServiceName:      "test-application-service",
						ServiceId:        "faeeeeee-aaaa-4011-0000-111111111111",
						ServiceClusterIp: "127.0.0.1",
						ServicePort:      80,
					},
				},
			},
		},
		StateTimestamp: int64(123456789),
		RiskScore:      3.14159,
	}

	expectedDeploymentJSON = `{
	"id": "deaaaaaa-bbbb-4011-0000-111111111111",
	"name": "Test Deployment",
	"hash": "12345678901",
	"type": "Deployment",
	"namespace": "Test Namespace",
	"namespaceId": "ccaaaaaa-bbbb-4011-0000-111111111111",
	"replicas": "1",
	"labels": {
		"app": "Test Application"
	},
	"podLabels": {
		"app": "Test Application"
	},
	"labelSelector": {
		"matchLabels": {
			"app": "Test Application"
		}
	},
	"created": "2020-12-24T23:59:59.999999999Z",
	"clusterId": "caaaaaaa-bbbb-4011-0000-111111111111",
	"clusterName": "Test Cluster",
	"containers": [
		{
			"id": "ccaaaaaa-cccc-4011-0000-111111111111:testapplication",
			"config": {},
			"image": {
				"id": "sha256:1234567890123456789012345678901234567890123456789012345678901234",
				"name": {
					"registry": "quay.io",
					"remote": "test/application",
					"tag": "demo",
					"fullName": "quay.io/test/application:demo"
				}
			},
			"securityContext": {},
			"ports": [
				{
					"containerPort": 80,
					"protocol": "TCP",
					"exposure": "INTERNAL",
					"exposureInfos": [
						{
							"level": "INTERNAL",
							"serviceName": "test-application-service",
							"serviceId": "faeeeeee-aaaa-4011-0000-111111111111",
							"serviceClusterIp": "127.0.0.1",
							"servicePort": 80
						}
					]
				}
			],
			"resources": {},
			"name": "Test Application",
			"livenessProbe": {},
			"readinessProbe": {}
		}
	],
	"priority": "42",
	"serviceAccount": "default",
	"serviceAccountPermissionLevel": "NONE",
	"automountServiceAccountToken": true,
	"ports": [
		{
			"containerPort": 80,
			"protocol": "TCP",
			"exposure": "INTERNAL",
			"exposureInfos": [
				{
					"level": "INTERNAL",
					"serviceName": "test-application-service",
					"serviceId": "faeeeeee-aaaa-4011-0000-111111111111",
					"serviceClusterIp": "127.0.0.1",
					"servicePort": 80
				}
			]
		}
	],
	"stateTimestamp": "123456789",
	"riskScore": 3.14159
}`
)

func (s *fakeDeploymentService) ExportDeployments(_ *v1.ExportDeploymentRequest, srv v1.DeploymentService_ExportDeploymentsServer) error {
	return srv.Send(&v1.ExportDeploymentResponse{Deployment: testDeployment})
}
