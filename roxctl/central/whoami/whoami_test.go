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
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestCentralWhoAmICommand(t *testing.T) {
	suite.Run(t, new(centralWhoAmITestSuite))
}

type centralWhoAmITestSuite struct {
	suite.Suite
	defaultCentralWhoAmICommand centralWhoAmICommand
}

type mockAuthServiceServer struct {
	v1.UnimplementedAuthServiceServer
	v1.UnimplementedRoleServiceServer

	userInfo         *storage.UserInfo
	resourceToAccess map[string]storage.Access
}

func (m *mockAuthServiceServer) GetAuthStatus(_ context.Context, _ *v1.Empty) (*v1.AuthStatus, error) {
	as := &v1.AuthStatus{}
	as.SetUserId(m.userInfo.GetUsername())
	as.SetUserInfo(m.userInfo)
	return as, nil
}

func (m *mockAuthServiceServer) GetMyPermissions(_ context.Context, _ *v1.Empty) (*v1.GetPermissionsResponse, error) {
	gpr := &v1.GetPermissionsResponse{}
	gpr.SetResourceToAccess(m.resourceToAccess)
	return gpr, nil
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
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
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
	c.Assert().Equal("UserID:\n\t\nUser name:\n\t\nRoles:\nAccess:\n", stdout.String())
}

func (c *centralWhoAmITestSuite) TestWhoIsHarald() {
	mockServer := &mockAuthServiceServer{
		userInfo: storage.UserInfo_builder{
			Username:     "Harald",
			FriendlyName: "Harald the second",
			Roles: []*storage.UserInfo_Role{
				storage.UserInfo_Role_builder{
					Name: "Warrior",
				}.Build(),
				storage.UserInfo_Role_builder{
					Name: "Engineer",
				}.Build(),
			},
		}.Build(),
		resourceToAccess: map[string]storage.Access{
			"Smartphone": storage.Access_READ_WRITE_ACCESS,
			"Library":    storage.Access_READ_ACCESS,
			"Valhalla":   storage.Access_NO_ACCESS,
		},
	}

	cbr, closeFunction, stdout, _ := c.setupCommand(mockServer)
	defer closeFunction()

	cbr.SetArgs([]string{"--timeout", "5s"})
	c.Require().NoError(cbr.Execute())
	c.Assert().Equal(`UserID:
	Harald
User name:
	Harald the second
Roles:
	- Warrior
	- Engineer
Access:
	r- Library
	rw Smartphone
	-- Valhalla
`,
		stdout.String())
}
