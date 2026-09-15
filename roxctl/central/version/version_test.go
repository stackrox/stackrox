package version

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version/productstreams"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stackrox/rox/pkg/version/versioncompatibility"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

type mockMetadataServer struct {
	v1.UnimplementedMetadataServiceServer
	version string
}

func (m *mockMetadataServer) GetMetadata(_ context.Context, _ *v1.Empty) (*v1.Metadata, error) {
	return &v1.Metadata{Version: m.version}, nil
}

func (c *centralVersionTestSuite) createGRPCMockService(server *mockMetadataServer) (*grpc.ClientConn, func()) {
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	srv := grpc.NewServer()
	v1.RegisterMetadataServiceServer(srv, server)

	go func() {
		utils.IgnoreError(func() error { return srv.Serve(listener) })
	}()

	conn, err := grpc.DialContext(context.Background(), "",
		grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	c.Require().NoError(err)

	return conn, func() {
		utils.IgnoreError(listener.Close)
		srv.Stop()
	}
}

func (c *centralVersionTestSuite) setupCommand(server *mockMetadataServer) (cmd *centralVersionCommand, stdout, stderr *bytes.Buffer, cleanup func()) {
	testutils.SetMainVersion(c.T(), "5.0.0-testing")
	conn, closeFunc := c.createGRPCMockService(server)
	env, out, errOut := mocks.NewEnvWithConn(conn, c.T())
	cmd = &centralVersionCommand{
		env:          env,
		timeout:      5 * time.Second,
		retryTimeout: 5 * time.Second,
	}
	return cmd, out, errOut, closeFunc
}

func (c *centralVersionTestSuite) TestCompatibilityStates() {
	productstreams.OverrideBumpsForTesting(c.T(), testBumpsYAML)

	tests := map[string]struct {
		centralVersion string
		wantCompat     versioncompatibility.Compatibility
		wantDisplay    string
	}{
		"matched": {
			centralVersion: "5.0.2",
			wantCompat:     versioncompatibility.Matched,
			wantDisplay:    "Matched",
		},
		"compatible ahead": {
			centralVersion: "5.3.1",
			wantCompat:     versioncompatibility.CompatibleAhead,
			wantDisplay:    "Compatible (Ahead)",
		},
		"compatible behind": {
			centralVersion: "4.10.1",
			wantCompat:     versioncompatibility.CompatibleBehind,
			wantDisplay:    "Compatible (Behind)",
		},
		"incompatible ahead": {
			centralVersion: "5.6.0",
			wantCompat:     versioncompatibility.IncompatibleAhead,
			wantDisplay:    "Incompatible (Ahead)",
		},
		"incompatible behind": {
			centralVersion: "4.5.2",
			wantCompat:     versioncompatibility.IncompatibleBehind,
			wantDisplay:    "Incompatible (Behind)",
		},
	}

	for name, tt := range tests {
		c.Run(name, func() {
			cmd, stdout, _, cleanup := c.setupCommand(&mockMetadataServer{version: tt.centralVersion})
			defer cleanup()

			err := cmd.run(false)

			c.Assert().NoError(err)

			output := stdout.String()
			c.Assert().Contains(output, "Central version:")
			c.Assert().Contains(output, tt.centralVersion)
			c.Assert().Contains(output, "roxctl version:")
			c.Assert().Contains(output, "5.0.0")
			c.Assert().Contains(output, "Compatible Central versions:")
			c.Assert().Contains(output, "Compatibility:")
			c.Assert().Contains(output, tt.wantDisplay)
			for line := range strings.SplitSeq(guidance(tt.wantCompat), "\n") {
				c.Assert().Contains(output, "  "+line)
			}
		})
	}
}

func (c *centralVersionTestSuite) TestEmptyVersionReturnsAuthError() {
	productstreams.OverrideBumpsForTesting(c.T(), testBumpsYAML)

	cmd, stdout, _, cleanup := c.setupCommand(&mockMetadataServer{version: ""})
	defer cleanup()

	err := cmd.run(false)

	c.Require().Error(err)
	c.Assert().Contains(err.Error(), "not authenticated")
	c.Assert().Empty(stdout.String())
}

func (c *centralVersionTestSuite) TestJSONOutput() {
	productstreams.OverrideBumpsForTesting(c.T(), testBumpsYAML)

	cmd, stdout, _, cleanup := c.setupCommand(&mockMetadataServer{version: "5.0.2"})
	defer cleanup()

	err := cmd.run(true)
	c.Require().NoError(err)

	var result versionResult
	c.Require().NoError(json.Unmarshal(stdout.Bytes(), &result))

	c.Assert().Equal("5.0.0-testing", result.RoxctlVersion)
	c.Assert().Equal("5.0.2", result.CentralVersion)
	c.Assert().Equal("MATCHED", result.Compatibility)
	c.Assert().NotEmpty(result.CompatibleCentralVersions)
	c.Assert().NotEmpty(result.Guidance)
}

func (c *centralVersionTestSuite) TestJSONOutputIncompatible() {
	productstreams.OverrideBumpsForTesting(c.T(), testBumpsYAML)

	cmd, stdout, _, cleanup := c.setupCommand(&mockMetadataServer{version: "5.6.0"})
	defer cleanup()

	err := cmd.run(true)

	c.Assert().NoError(err)

	var result versionResult
	c.Require().NoError(json.Unmarshal(stdout.Bytes(), &result))

	c.Assert().Equal("INCOMPATIBLE_AHEAD", result.Compatibility)
	c.Assert().Equal("5.6.0", result.CentralVersion)
}

func (c *centralVersionTestSuite) TestTextOutputFormat() {
	productstreams.OverrideBumpsForTesting(c.T(), testBumpsYAML)

	cmd, stdout, _, cleanup := c.setupCommand(&mockMetadataServer{version: "5.0.2"})
	defer cleanup()

	err := cmd.run(false)
	c.Require().NoError(err)

	output := stdout.String()
	c.Assert().Contains(output, "Central version:             5.0.2")
	c.Assert().Contains(output, "roxctl version:              5.0.0")
	c.Assert().Contains(output, "Compatibility:               Matched")
	c.Assert().Contains(output, "  Compatible Central versions:")
	c.Assert().Contains(output, "  roxctl version is matched with Central.")
}

func TestDisplayName(t *testing.T) {
	tests := map[string]struct {
		c    versioncompatibility.Compatibility
		want string
	}{
		"matched":             {versioncompatibility.Matched, "Matched"},
		"compatible behind":   {versioncompatibility.CompatibleBehind, "Compatible (Behind)"},
		"compatible ahead":    {versioncompatibility.CompatibleAhead, "Compatible (Ahead)"},
		"incompatible behind": {versioncompatibility.IncompatibleBehind, "Incompatible (Behind)"},
		"incompatible ahead":  {versioncompatibility.IncompatibleAhead, "Incompatible (Ahead)"},
		"unknown":             {versioncompatibility.Unknown, "Unknown"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, displayName(tt.c))
		})
	}
}

func TestGuidance(t *testing.T) {
	tests := map[string]struct {
		c         versioncompatibility.Compatibility
		wantEmpty bool
		contains  string
	}{
		"matched":             {versioncompatibility.Matched, false, "matched with Central"},
		"compatible ahead":    {versioncompatibility.CompatibleAhead, false, "ahead of roxctl"},
		"compatible behind":   {versioncompatibility.CompatibleBehind, false, "behind roxctl"},
		"incompatible ahead":  {versioncompatibility.IncompatibleAhead, false, "outside the compatible version range"},
		"incompatible behind": {versioncompatibility.IncompatibleBehind, false, "outside the compatible version range"},
		"unknown":             {versioncompatibility.Unknown, true, ""},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			g := guidance(tt.c)
			if tt.wantEmpty {
				assert.Empty(t, g)
			} else {
				require.NotEmpty(t, g)
				assert.Contains(t, g, tt.contains)
			}
		})
	}
}
