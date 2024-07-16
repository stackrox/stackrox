package deployments

import (
	"bytes"
	"context"
	"testing"
	"time"

	// Embed is used to import the serialized test object file.
	_ "embed"

	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fixtures"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/protocompat"
	envMocks "github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stackrox/rox/roxctl/common/flags"
	ioMocks "github.com/stackrox/rox/roxctl/common/io/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
)

//go:embed serialized_test_deployment.json
var expectedJSONSerializedDeployment string

func TestExportDeployments(t *testing.T) {
	fakeService := &fakeDeploymentService{tb: t}
	conn, closeFunc, err := pkgGRPC.CreateTestGRPCStreamingService(
		context.Background(),
		t,
		func(registrar grpc.ServiceRegistrar) {
			v1.RegisterDeploymentServiceServer(registrar, fakeService)
		},
	)
	require.NoError(t, err)
	defer closeFunc()

	mockCtrl := gomock.NewController(t)

	var buf bytes.Buffer
	ioMock := ioMocks.NewMockIO(mockCtrl)
	ioMock.EXPECT().Out().Times(1).Return(&buf)

	env := envMocks.NewMockEnvironment(mockCtrl)
	env.EXPECT().GRPCConnection().Times(1).Return(conn, nil)
	env.EXPECT().InputOutput().Times(1).Return(ioMock)

	fakeCmd := &cobra.Command{}
	flags.AddTimeoutWithDefault(fakeCmd, 10*time.Second)

	cmd := Command(env)
	err = cmd.RunE(fakeCmd, []string{})
	assert.NoError(t, err)

	assert.JSONEq(t, `{"deployment":`+expectedJSONSerializedDeployment+`}`, buf.String())
}

type fakeDeploymentService struct {
	tb testing.TB
}

func (s *fakeDeploymentService) ExportDeployments(_ *v1.ExportDeploymentRequest, srv v1.DeploymentService_ExportDeploymentsServer) error {
	var testDeployment = getDeploymentForSerializationTest()
	return srv.Send(&v1.ExportDeploymentResponse{Deployment: testDeployment})
}

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

// getDeploymentForSerializationTest returns a Mock Deployment for serialization testing purpose.
func getDeploymentForSerializationTest() *storage.Deployment {
	deployment := fixtures.LightweightDeployment()
	var createdDate = time.Date(2020, time.December, 24, 23, 59, 59, 999999999, time.UTC)
	deployment.Created = protocompat.ConvertTimeToTimestampOrNil(&createdDate)
	return deployment
}
