package whoami

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
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/mocks"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

func TestCentralWhoAmICommand(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(centralWhoAmITestSuite))
}

type centralWhoAmITestSuite struct {
	suite.Suite
	defaultCentralWhoAmICommand centralWhoAmICommand
}

type mockAuthServiceServer struct {
	v1.AuthServiceServer
	v1.RoleServiceServer

	userInfo         *storage.UserInfo
	resourceToAccess map[string]storage.Access
	roles            []*storage.Role
}

func (m *mockAuthServiceServer) GetAuthStatus(ctx context.Context, req *v1.Empty) (*v1.AuthStatus, error) {
	return &v1.AuthStatus{
		Id: &v1.AuthStatus_UserId{
			UserId: m.userInfo.Username,
		},
		UserInfo: m.userInfo,
	}, nil
}

func (m *mockAuthServiceServer) GetMyPermissions(ctx context.Context, req *v1.Empty) (*v1.GetPermissionsResponse, error) {
	return &v1.GetPermissionsResponse{ResourceToAccess: m.resourceToAccess}, nil
}

func (m *mockAuthServiceServer) GetRoles(ctx context.Context, req *v1.Empty) (*v1.GetRolesResponse, error) {
	return &v1.GetRolesResponse{Roles: m.roles}, nil
}

func (c *centralWhoAmITestSuite) createGRPCMockServices(mockServer *mockAuthServiceServer) (*grpc.ClientConn, func()) {
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	server := grpc.NewServer()

	v1.RegisterAuthServiceServer(server, mockServer)
	v1.RegisterRoleServiceServer(server, mockServer)

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

func (c *centralWhoAmITestSuite) SetupTest() {
	c.defaultCentralWhoAmICommand = centralWhoAmICommand{
		timeout: 5 * time.Second,
	}
	os.Stderr = nil
}

func (c *centralWhoAmITestSuite) setupCommand(mockServer *mockAuthServiceServer) (*cobra.Command, func(), *bytes.Buffer, *bytes.Buffer) {
	conn, closeFunction := c.createGRPCMockServices(mockServer)
	mockedEnv, stdout, stderr := mocks.NewEnvWithConn(conn, c.T())
	cbr := Command(mockedEnv)
	return cbr, closeFunction, stdout, stderr
}

func (c *centralWhoAmITestSuite) TestWhoAmIEmpty() {
	mockServer := &mockAuthServiceServer{userInfo: &storage.UserInfo{}}

	cbr, closeFunction, stdout, _ := c.setupCommand(mockServer)
	defer closeFunction()

	cbr.SetArgs([]string{"--timeout", "5s"})
	c.Require().NoError(cbr.Execute())
	c.Assert().Equal(stdout.String(), "User:\n  \nRoles:\n \nAccess:\n")
}

func (c *centralWhoAmITestSuite) TestWhoIsHarald() {
	mockServer := &mockAuthServiceServer{
		userInfo: &storage.UserInfo{Username: "Harald"},
		resourceToAccess: map[string]storage.Access{
			"Smartphone": storage.Access_READ_WRITE_ACCESS,
			"Library":    storage.Access_READ_ACCESS,
			"Valhalla":   storage.Access_NO_ACCESS,
		},
		roles: []*storage.Role{{Name: "Warrior"}, {Name: "Engineer"}},
	}

	cbr, closeFunction, stdout, _ := c.setupCommand(mockServer)
	defer closeFunction()

	cbr.SetArgs([]string{"--timeout", "5s"})
	c.Require().NoError(cbr.Execute())
	c.Assert().Equal(stdout.String(),
		"User:\n  Harald\nRoles:\n Warrior, Engineer\nAccess:\n  r- Library\n  rw Smartphone\n  -- Valhalla\n")
}
