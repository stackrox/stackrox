package delete

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"

	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

type mockClustersServiceServer struct {
	v1.ClustersServiceServer
	clusters []*storage.Cluster
}

func (m *mockClustersServiceServer) GetClusters(ctx context.Context, req *v1.GetClustersRequest) (*v1.ClustersList, error) {
	return &v1.ClustersList{Clusters: m.clusters}, nil
}

func (m *mockClustersServiceServer) DeleteCluster(ctx context.Context, req *v1.ResourceByID) (*v1.Empty, error) {
	return &v1.Empty{}, nil
}

func TestClusterDeleteCommand(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(clusterDeleteTestSuite))
}

type clusterDeleteTestSuite struct {
	suite.Suite
	defaultClusterDeleteCommand clusterDeleteCommand
}

func (c *clusterDeleteTestSuite) createGRPCMockClustersService(clusters []*storage.Cluster) (*grpc.ClientConn, func()) {
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	server := grpc.NewServer()
	v1.RegisterClustersServiceServer(server, &mockClustersServiceServer{clusters: clusters})

	go func() {
		utils.IgnoreError(func() error { return server.Serve(listener) })
	}()

	conn, err := grpc.DialContext(context.Background(), "", grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithInsecure())
	c.Require().NoError(err)

	closeFunction := func() {
		utils.IgnoreError(listener.Close)
		server.Stop()
	}

	return conn, closeFunction
}

func (c *clusterDeleteTestSuite) createMockEnvironmentWithConn(conn *grpc.ClientConn) (environment.Environment, *bytes.Buffer, *bytes.Buffer) {
	return mocks.NewEnvWithConn(conn, c.T())
}

func (c *clusterDeleteTestSuite) SetupTest() {
	c.defaultClusterDeleteCommand = clusterDeleteCommand{
		name:    "dummy",
		timeout: 5 * time.Second,
	}
}

func (c *clusterDeleteTestSuite) TestConstructSetsTimeoutFlag() {
	expectedTimeout := 10 * time.Minute
	testCmd := &cobra.Command{Use: "test"}
	testCmd.Flags().Duration("timeout", expectedTimeout, "")

	clusterDeleteCmd := c.defaultClusterDeleteCommand
	err := clusterDeleteCmd.Construct(nil, testCmd)

	c.Assert().NoError(err)
	c.Assert().Equal(clusterDeleteCmd.timeout, expectedTimeout, "Timeout does not match value of '--timeout' flag.")
}

func (c *clusterDeleteTestSuite) TestValidateRequiresName() {
	clusterDeleteCmd := c.defaultClusterDeleteCommand
	clusterDeleteCmd.name = ""

	err := clusterDeleteCmd.Validate()

	c.Require().Error(err)
	c.Assert().ErrorIs(err, errNameIsRequired)
}

func (c *clusterDeleteTestSuite) TestDeleteHappyPath() {
	clusters := []*storage.Cluster{{Name: "dummy"}}
	conn, closeFunction := c.createGRPCMockClustersService(clusters)
	defer closeFunction()
	clusterDeleteCmd := c.defaultClusterDeleteCommand
	mockedEnv, stdout, _ := c.createMockEnvironmentWithConn(conn)
	clusterDeleteCmd.env = mockedEnv

	err := clusterDeleteCmd.Validate()
	c.Require().NoError(err)

	err = clusterDeleteCmd.Delete()
	c.Require().NoError(err)
	c.Assert().Contains(stdout.String(), `Successfully deleted cluster "dummy"`)
}

func (c *clusterDeleteTestSuite) TestDeleteFailsIfClusterNotFound() {
	clusters := []*storage.Cluster{}
	conn, closeFunction := c.createGRPCMockClustersService(clusters)
	defer closeFunction()
	clusterDeleteCmd := c.defaultClusterDeleteCommand
	clusterDeleteCmd.env, _, _ = c.createMockEnvironmentWithConn(conn)

	err := clusterDeleteCmd.Validate()
	c.Require().NoError(err)

	err = clusterDeleteCmd.Delete()
	c.Require().Error(err)
}
