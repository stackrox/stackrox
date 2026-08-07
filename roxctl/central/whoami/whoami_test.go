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
	"github.com/stackrox/rox/pkg/version/productstreams"
	versiontestutils "github.com/stackrox/rox/pkg/version/testutils"
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
	v1.UnimplementedMetadataServiceServer

	userInfo         *storage.UserInfo
	resourceToAccess map[string]storage.Access
	centralVersion   string
}

func (m *mockAuthServiceServer) GetAuthStatus(_ context.Context, _ *v1.Empty) (*v1.AuthStatus, error) {
	return &v1.AuthStatus{
		Id: &v1.AuthStatus_UserId{
			UserId: m.userInfo.GetUsername(),
		},
		UserInfo: m.userInfo,
	}, nil
}

func (m *mockAuthServiceServer) GetMyPermissions(_ context.Context, _ *v1.Empty) (*v1.GetPermissionsResponse, error) {
	return &v1.GetPermissionsResponse{ResourceToAccess: m.resourceToAccess}, nil
}

func (m *mockAuthServiceServer) GetMetadata(_ context.Context, _ *v1.Empty) (*v1.Metadata, error) {
	return &v1.Metadata{Version: m.centralVersion}, nil
}

func (c *centralWhoAmITestSuite) createGRPCMockServices(mockServer *mockAuthServiceServer) (*grpc.ClientConn, func()) {
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	server := grpc.NewServer()

	v1.RegisterAuthServiceServer(server, mockServer)
	v1.RegisterRoleServiceServer(server, mockServer)
	v1.RegisterMetadataServiceServer(server, mockServer)

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
		userInfo: &storage.UserInfo{
			Username:     "Harald",
			FriendlyName: "Harald the second",
			Roles: []*storage.UserInfo_Role{
				{
					Name: "Warrior",
				},
				{
					Name: "Engineer",
				},
			},
		},
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

func (c *centralWhoAmITestSuite) setupVersionTest(centralVersion string) (*cobra.Command, func(), *bytes.Buffer, *bytes.Buffer) {
	const testBumpsYAML = `bumps:
  - from: "3.74"
    to: "4.0"
  - from: "4.11"
    to: "5.0"
`
	versiontestutils.SetMainVersion(c.T(), "5.0.0")
	productstreams.OverrideBumpsForTesting(c.T(), testBumpsYAML)

	mockServer := &mockAuthServiceServer{
		userInfo:       &storage.UserInfo{Username: "user", FriendlyName: "user"},
		centralVersion: centralVersion,
	}
	return c.setupCommand(mockServer)
}

func (c *centralWhoAmITestSuite) TestVersionCompatibilityMatched() {
	cbr, closeFunction, _, stderr := c.setupVersionTest("5.0.2")
	defer closeFunction()

	cbr.SetArgs([]string{"--timeout", "5s"})
	c.Require().NoError(cbr.Execute())
	c.Assert().Empty(stderr.String())
}

func (c *centralWhoAmITestSuite) TestVersionCompatibilityCompatibleAhead() {
	cbr, closeFunction, _, stderr := c.setupVersionTest("5.3.1")
	defer closeFunction()

	cbr.SetArgs([]string{"--timeout", "5s"})
	c.Require().NoError(cbr.Execute())
	c.Assert().Empty(stderr.String())
}

func (c *centralWhoAmITestSuite) TestVersionCompatibilityCompatibleBehind() {
	cbr, closeFunction, _, stderr := c.setupVersionTest("4.10.1")
	defer closeFunction()

	cbr.SetArgs([]string{"--timeout", "5s"})
	c.Require().NoError(cbr.Execute())
	c.Assert().Empty(stderr.String())
}

func (c *centralWhoAmITestSuite) TestVersionCompatibilityIncompatibleAhead() {
	cbr, closeFunction, _, stderr := c.setupVersionTest("5.6.0")
	defer closeFunction()

	cbr.SetArgs([]string{"--timeout", "5s"})
	c.Require().NoError(cbr.Execute())
	c.Assert().Contains(stderr.String(), "Warning: Your roxctl 5.0.0 is too old for this Central 5.6.0.")
	c.Assert().Contains(stderr.String(), "roxctl: 5.0.0 | Central: 5.6.0 | Compatible Centrals: 4.9, 4.10, 4.11, 5.0, 5.1, 5.2, 5.3")
}

func (c *centralWhoAmITestSuite) TestVersionCompatibilityIncompatibleBehind() {
	cbr, closeFunction, _, stderr := c.setupVersionTest("4.5.2")
	defer closeFunction()

	cbr.SetArgs([]string{"--timeout", "5s"})
	c.Require().NoError(cbr.Execute())
	c.Assert().Contains(stderr.String(), "Warning: Your roxctl 5.0.0 is too new for this Central 4.5.2.")
	c.Assert().Contains(stderr.String(), "roxctl: 5.0.0 | Central: 4.5.2 | Compatible Centrals: 4.9, 4.10, 4.11, 5.0, 5.1, 5.2, 5.3")
}

func (c *centralWhoAmITestSuite) TestVersionCompatibilityUnknownCentralVersion() {
	cbr, closeFunction, _, stderr := c.setupVersionTest("")
	defer closeFunction()

	cbr.SetArgs([]string{"--timeout", "5s"})
	c.Require().NoError(cbr.Execute())
	c.Assert().Contains(stderr.String(), "WARN:")
	c.Assert().Contains(stderr.String(), "Central did not report its version")
}
