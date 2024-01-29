package delete

import (
	"bytes"
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestClusterDeleteCommand(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(clusterDeleteTestSuite))
}

type clusterDeleteTestSuite struct {
	suite.Suite
	defaultClusterDeleteCommand clusterDeleteCommand
}

type mockClustersServiceServer struct {
	v1.UnimplementedClustersServiceServer

	clusters []*storage.Cluster
}

func (m *mockClustersServiceServer) GetClusters(_ context.Context, _ *v1.GetClustersRequest) (*v1.ClustersList, error) {
	return &v1.ClustersList{Clusters: m.clusters}, nil
}

func (m *mockClustersServiceServer) DeleteCluster(_ context.Context, _ *v1.ResourceByID) (*v1.Empty, error) {
	return &v1.Empty{}, nil
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
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	c.Require().NoError(err)

	closeFunction := func() {
		utils.IgnoreError(listener.Close)
		server.Stop()
	}

	return conn, closeFunction
}

func (c *clusterDeleteTestSuite) SetupTest() {
	c.defaultClusterDeleteCommand = clusterDeleteCommand{
		name:         "dummy",
		timeout:      5 * time.Second,
		retryTimeout: 5 * time.Second,
	}
	os.Stderr = nil
}

func (c *clusterDeleteTestSuite) setupCommand(clusters []*storage.Cluster) (*cobra.Command, func(), *bytes.Buffer, *bytes.Buffer) {
	conn, closeFunction := c.createGRPCMockClustersService(clusters)
	mockedEnv, stdout, stderr := mocks.NewEnvWithConn(conn, c.T())
	cbr := Command(mockedEnv)
	cbr.PersistentFlags().DurationP("timeout", "t", 5*time.Second, "timeout for API requests")
	cbr.PersistentFlags().Duration("retry-timeout", 5*time.Second, "retry timeout for API requests")

	return cbr, closeFunction, stdout, stderr
}

func (c *clusterDeleteTestSuite) TestCommandHappyPath() {
	clusters := []*storage.Cluster{{Name: "dummy"}}
	cbr, closeFunction, stdout, _ := c.setupCommand(clusters)
	defer closeFunction()

	cbr.SetArgs([]string{"--name", "dummy"})
	err := cbr.Execute()

	c.Require().NoError(err)
	c.Assert().Contains(stdout.String(), `Successfully deleted cluster "dummy"`)
}

func (c *clusterDeleteTestSuite) TestCommandRequiresName() {
	clusters := []*storage.Cluster{{Name: "dummy"}}
	cbr, closeFunction, _, _ := c.setupCommand(clusters)
	defer closeFunction()

	err := cbr.Execute()

	c.Require().Error(err)
	c.Assert().ErrorIs(err, errox.InvalidArgs)
}

func (c *clusterDeleteTestSuite) TestCommandFailsIfClusterNotFound() {
	clusters := []*storage.Cluster{}
	cbr, closeFunction, _, _ := c.setupCommand(clusters)
	defer closeFunction()

	cbr.SetArgs([]string{"--name", "dummy"})
	err := cbr.Execute()

	c.Require().Error(err)
	c.Assert().ErrorIs(err, errox.NotFound)
}

func (c *clusterDeleteTestSuite) TestConstructSetsTimeoutFlag() {
	expectedTimeout := 10 * time.Minute
	testCmd := &cobra.Command{Use: "test"}
	testCmd.Flags().Duration("timeout", expectedTimeout, "")
	testCmd.Flags().Duration("retry-timeout", expectedTimeout, "")

	clusterDeleteCmd := c.defaultClusterDeleteCommand
	err := clusterDeleteCmd.Construct(nil, testCmd)

	c.Assert().NoError(err)
	c.Assert().Equal(clusterDeleteCmd.timeout, expectedTimeout, "Timeout does not match value of '--timeout' flag.")
	c.Assert().Equal(clusterDeleteCmd.retryTimeout, expectedTimeout, "Retry timeout does not match value of '--retry-timeout' flag.")
}
