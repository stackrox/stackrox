package pods

import (
	"bytes"
	"context"
	"testing"
	"time"

	// Embed is used to import the serialized test object file.
	_ "embed"

	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fixtures"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	envMocks "github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stackrox/rox/roxctl/common/flags"
	ioMocks "github.com/stackrox/rox/roxctl/common/io/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
)

//go:embed serialized_test_pod.json
var expectedJSONSerializedPod string

func TestExportPods(t *testing.T) {
	fakeService := &fakePodsService{tb: t}
	conn, closeFunc, err := pkgGRPC.CreateTestGRPCStreamingService(
		context.Background(),
		t,
		func(registrar grpc.ServiceRegistrar) {
			v1.RegisterPodServiceServer(registrar, fakeService)
		},
	)
	require.NoError(t, err)
	defer closeFunc()

	mockCtrl := gomock.NewController(t)
	var buf bytes.Buffer

	mockIO := ioMocks.NewMockIO(mockCtrl)
	mockIO.EXPECT().Out().Times(1).Return(&buf)

	mockEnv := envMocks.NewMockEnvironment(mockCtrl)
	mockEnv.EXPECT().GRPCConnection().Times(1).Return(conn, nil)
	mockEnv.EXPECT().InputOutput().Times(1).Return(mockIO)

	fakeCmd := &cobra.Command{}
	flags.AddTimeoutWithDefault(fakeCmd, 10*time.Second)

	cmd := Command(mockEnv)
	err = cmd.RunE(fakeCmd, []string{})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"pod":`+expectedJSONSerializedPod+`}`, buf.String())
}

type fakePodsService struct {
	tb testing.TB
}

func (s *fakePodsService) ExportPods(_ *v1.ExportPodRequest, srv v1.PodService_ExportPodsServer) error {
	testPod := fixtures.GetPod()
	return srv.Send(&v1.ExportPodResponse{Pod: testPod})
}

func (s *fakePodsService) GetPods(_ context.Context, _ *v1.RawQuery) (*v1.PodsResponse, error) {
	return nil, errox.NotImplemented
}
