package nodes

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
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

func TestExportNodes(t *testing.T) {
	fakeService := &fakeNodeService{tb: t}
	conn, closeFunc, err := pkgGRPC.CreateTestGRPCStreamingService(
		context.Background(),
		t,
		func(registrar grpc.ServiceRegistrar) {
			v1.RegisterNodeServiceServer(registrar, fakeService)
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
	expectedSerializedNode := fixtures.GetExpectedJSONSerializedTestNode(t)
	assert.JSONEq(t, `{"node":`+expectedSerializedNode+`}`, buf.String())
}

type fakeNodeService struct {
	tb testing.TB
}

func (s *fakeNodeService) ExportNodes(_ *v1.ExportNodeRequest, srv v1.NodeService_ExportNodesServer) error {
	testNode := fixtures.GetNodeForSerializationTest(s.tb)
	return srv.Send(&v1.ExportNodeResponse{Node: testNode})
}

func (s *fakeNodeService) ListNodes(_ context.Context, _ *v1.ListNodesRequest) (*v1.ListNodesResponse, error) {
	return nil, errox.NotImplemented
}

func (s *fakeNodeService) GetNode(_ context.Context, _ *v1.GetNodeRequest) (*storage.Node, error) {
	return nil, errox.NotImplemented
}

/*

	if err := srv.Send(&v1.ExportNodeResponse{Node: node}); err != nil {
		return err
	}

*/
