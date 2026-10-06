package version

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version/productstreams"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stackrox/rox/pkg/version/versioncompatibility"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stackrox/rox/roxctl/common/versioncheck"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

const testBumpsYAML = `bumps:
  - from: "3.74"
    to: "4.0"
  - from: "4.11"
    to: "5.0"
`

func TestCentralVersionCommand(t *testing.T) {
	suite.Run(t, new(centralVersionTestSuite))
}

type centralVersionTestSuite struct {
	suite.Suite
}

func (c *centralVersionTestSuite) TearDownTest() {
	versioncheck.UnsuppressVersionMismatchWarningForTesting(c.T())
}

func (c *centralVersionTestSuite) TestCompatibilityStates() {
	productstreams.OverrideBumpsForTesting(c.T(), testBumpsYAML)

	tests := map[string]struct {
		centralVersion string
		wantCompat     versioncompatibility.Compatibility
		wantDisplay    string
		wantContains   []string
	}{
		"matched": {
			centralVersion: "5.0.2",
			wantCompat:     versioncompatibility.Matched,
			wantDisplay:    "Matched",
			wantContains:   []string{"matched with Central"},
		},
		"compatible ahead": {
			centralVersion: "5.3.1",
			wantCompat:     versioncompatibility.CompatibleAhead,
			wantDisplay:    "Compatible (Ahead)",
			wantContains:   []string{"ahead of roxctl", "Use newer roxctl"},
		},
		"compatible behind": {
			centralVersion: "4.10.1",
			wantCompat:     versioncompatibility.CompatibleBehind,
			wantDisplay:    "Compatible (Behind)",
			wantContains:   []string{"behind roxctl", "Central upgrade"},
		},
		"incompatible ahead": {
			centralVersion: "5.6.0",
			wantCompat:     versioncompatibility.IncompatibleAhead,
			wantDisplay:    "Incompatible (Ahead)",
			wantContains:   []string{"ahead of roxctl", "Use newer roxctl"},
		},
		"incompatible behind": {
			centralVersion: "4.5.2",
			wantCompat:     versioncompatibility.IncompatibleBehind,
			wantDisplay:    "Incompatible (Behind)",
			wantContains:   []string{"behind roxctl", "Central upgrade"},
		},
	}

	for name, tt := range tests {
		c.Run(name, func() {
			cmd, stdout, _, interceptorOutput, cleanup := c.setupCommand(&mockMetadataServer{version: tt.centralVersion})
			defer cleanup()

			err := cmd.run(false)

			c.Assert().NoError(err)
			c.Assert().Empty(interceptorOutput.String(), "version check interceptor warning should be suppressed")

			output := stdout.String()
			c.Assert().Contains(output, "Central version:")
			c.Assert().Contains(output, tt.centralVersion)
			c.Assert().Contains(output, "roxctl version:")
			c.Assert().Contains(output, "5.0.0")
			c.Assert().Contains(output, "Compatible Central versions:")
			c.Assert().Contains(output, "Compatibility:")
			c.Assert().Contains(output, tt.wantDisplay)
			for _, s := range tt.wantContains {
				c.Assert().Contains(output, s)
			}
		})
	}
}

func (c *centralVersionTestSuite) TestEmptyVersionReturnsAuthError() {
	productstreams.OverrideBumpsForTesting(c.T(), testBumpsYAML)

	cmd, stdout, _, _, cleanup := c.setupCommand(&mockMetadataServer{version: ""})
	defer cleanup()

	err := cmd.run(false)

	c.Require().Error(err)
	c.Assert().Contains(err.Error(), "not authenticated")
	c.Assert().Empty(stdout.String())
}

func (c *centralVersionTestSuite) TestJSONOutput() {
	productstreams.OverrideBumpsForTesting(c.T(), testBumpsYAML)

	cmd, stdout, _, _, cleanup := c.setupCommand(&mockMetadataServer{version: "5.0.2"})
	defer cleanup()

	err := cmd.run(true)
	c.Require().NoError(err)

	var result versioncheck.VersionCheckResult
	c.T().Log(stdout.String())
	c.Require().NoError(json.Unmarshal(stdout.Bytes(), &result))

	c.Assert().Equal("5.0.0-testing", result.RoxctlVersion)
	c.Assert().Equal("5.0.2", result.CentralVersion)
	c.Assert().Equal("MATCHED", result.Compatibility)
	c.Assert().NotEmpty(result.CompatibleCentralVersions)
	c.Assert().NotEmpty(result.Summary)
}

func (c *centralVersionTestSuite) TestJSONOutputIncompatible() {
	productstreams.OverrideBumpsForTesting(c.T(), testBumpsYAML)

	cmd, stdout, _, interceptorOutput, cleanup := c.setupCommand(&mockMetadataServer{version: "5.6.0"})
	defer cleanup()

	err := cmd.run(true)

	c.Assert().NoError(err)
	c.Assert().Empty(interceptorOutput.String(), "version check interceptor warning should be suppressed")

	var result versioncheck.VersionCheckResult
	c.Require().NoError(json.Unmarshal(stdout.Bytes(), &result))

	c.Assert().Equal("INCOMPATIBLE_AHEAD", result.Compatibility)
	c.Assert().Equal("5.6.0", result.CentralVersion)
}

func (c *centralVersionTestSuite) TestTextOutputFormat() {
	productstreams.OverrideBumpsForTesting(c.T(), testBumpsYAML)

	cmd, stdout, _, _, cleanup := c.setupCommand(&mockMetadataServer{version: "5.0.2"})
	defer cleanup()

	err := cmd.run(false)
	c.Require().NoError(err)

	output := stdout.String()
	c.Assert().Contains(output, "Central version:                   5.0.2")
	c.Assert().Contains(output, "roxctl version:                    5.0.0")
	c.Assert().Contains(output, "Compatibility:                     Matched")
	c.Assert().Contains(output, "Compatible Central versions:")
	c.Assert().Contains(output, "  roxctl version is matched with Central.")
}

// --- helpers and mocks ---

type mockMetadataServer struct {
	v1.UnimplementedMetadataServiceServer
	version string
}

func (m *mockMetadataServer) GetMetadata(_ context.Context, _ *v1.Empty) (*v1.Metadata, error) {
	return &v1.Metadata{Version: m.version}, nil
}

func (c *centralVersionTestSuite) createGRPCMockService(server *mockMetadataServer) (*grpc.ClientConn, *bytes.Buffer, func()) {
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(injectVersionHeader(server.version)))
	v1.RegisterMetadataServiceServer(srv, server)

	go func() {
		utils.IgnoreError(func() error { return srv.Serve(listener) })
	}()

	var interceptorOutput bytes.Buffer
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(versioncheck.CentralVersionClientInterceptor(&interceptorOutput)),
	)
	c.Require().NoError(err)

	return conn, &interceptorOutput, func() {
		utils.IgnoreError(conn.Close)
		utils.IgnoreError(listener.Close)
		srv.Stop()
	}
}

func (c *centralVersionTestSuite) setupCommand(server *mockMetadataServer) (cmd *centralVersionCommand, stdout, stderr, interceptorOutput *bytes.Buffer, cleanup func()) {
	testutils.SetMainVersion(c.T(), "5.0.0-testing")
	conn, interceptorOut, closeFunc := c.createGRPCMockService(server)
	env, out, errOut := mocks.NewEnvWithConn(conn, c.T())
	cmd = &centralVersionCommand{
		env:          env,
		timeout:      5 * time.Second,
		retryTimeout: 5 * time.Second,
	}
	return cmd, out, errOut, interceptorOut, closeFunc
}

func injectVersionHeader(centralVersion string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		_ = grpc.SetHeader(ctx, metadata.Pairs(clientconn.CentralVersionHeader, centralVersion))
		return handler(ctx, req)
	}
}
