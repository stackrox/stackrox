package crs

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestCentralCrsGenerateCommand(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(centralCrsGenerateTestSuite))
}

type centralCrsGenerateTestSuite struct {
	suite.Suite
}

type mockClusterInitServiceServer struct {
	v1.UnimplementedClusterInitServiceServer
	suite.Suite
}

func (m *mockClusterInitServiceServer) computeExpiresAt(req *v1.CRSGenRequestExtended) *timestamppb.Timestamp {

}

func (m *mockClusterInitServiceServer) GenerateCRSExtended(_ context.Context, req *v1.CRSGenRequestExtended) (*v1.CRSGenResponse, error) {
	// m.T().Logf("Request: %v", req)
	fmt.Fprintf(os.Stderr, "Request: %v\n", req)
	return &v1.CRSGenResponse{
		Meta: &v1.CRSMeta{
			Id:        "8c2bfcd4-d554-4235-84b0-5ee39ab3926c",
			Name:      req.GetName(),
			CreatedAt: timestamppb.New(time.Now()),
			CreatedBy: &storage.User{
				Id: "roxctl-test-suite",
			},
			ExpiresAt: req.GetValidUntil(),
		},
		Crs: []byte("dummy"),
	}, nil
}

func (c *centralCrsGenerateTestSuite) createGRPCMockServices(mockServer *mockClusterInitServiceServer) (*grpc.ClientConn, func()) {
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	server := grpc.NewServer()

	v1.RegisterClusterInitServiceServer(server, mockServer)

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

func (c *centralCrsGenerateTestSuite) SetupTest() {
	// os.Stderr = nil
}

func (c *centralCrsGenerateTestSuite) setupCommand(mockServer *mockClusterInitServiceServer) (*cobra.Command, func(), *bytes.Buffer, *bytes.Buffer) {
	conn, closeFunction := c.createGRPCMockServices(mockServer)
	mockedEnv, stdout, stderr := mocks.NewEnvWithConn(conn, c.T())
	cbr := generateCommand(mockedEnv)
	return cbr, closeFunction, stdout, stderr
}

func (c *centralCrsGenerateTestSuite) TestCrsGenerateExtended() {
	mockServer := &mockClusterInitServiceServer{Suite: c.Suite}

	cbr, closeFunction, stdout, stderr := c.setupCommand(mockServer)
	defer closeFunction()

	cbr.SetArgs([]string{"--timeout=5s", "secured-cluster-name", "--output", "-", "--valid-for=10m"})
	c.Require().NoError(cbr.Execute())
	c.Assert().NotEmpty(stdout.String())

	fmt.Println(stderr.String())
	// c.Assert().Equal("UserID:\n\t\nUser name:\n\t\nRoles:\nAccess:\n", stdout.String())
}
