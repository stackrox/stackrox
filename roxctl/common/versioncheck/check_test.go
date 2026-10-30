package versioncheck

import (
	"bytes"
	"context"
	"net"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/versionheader"
	"github.com/stackrox/rox/pkg/version/productstreams"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stackrox/rox/pkg/version/versioncompatibility"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestCheckAndWarn(t *testing.T) {
	cases := map[string]struct {
		localVersion   string
		centralVersion string
		expectWarning  string
	}{
		"same version": {
			localVersion:   "4.8.0",
			centralVersion: "4.8.0",
		},
		"same XY different patch": {
			localVersion:   "4.8.0",
			centralVersion: "4.8.1",
		},
		"compatible behind within range": {
			localVersion:   "4.8.0",
			centralVersion: "4.6.0",
		},
		"compatible ahead within range": {
			localVersion:   "4.8.0",
			centralVersion: "4.10.0",
		},
		"incompatible behind": {
			localVersion:   "4.8.0",
			centralVersion: "4.3.0",
			expectWarning:  "Plan a Central upgrade",
		},
		"incompatible ahead": {
			localVersion:   "4.8.0",
			centralVersion: "4.15.0",
			expectWarning:  "Use newer roxctl version",
		},
		"invalid central version - no warning": {
			localVersion:   "4.8.0",
			centralVersion: "invalid",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			testutils.SetMainVersion(t, tc.localVersion)

			var buf bytes.Buffer
			checkAndWarn(tc.centralVersion, &buf)

			if tc.expectWarning != "" {
				assert.Contains(t, buf.String(), tc.expectWarning)
				assert.Contains(t, buf.String(), "Compatible Centrals:")
			} else {
				assert.Empty(t, buf.String())
			}
		})
	}
}

func TestCheckAndWarnPanicsOnInvalidLocalVersion(t *testing.T) {
	var buf bytes.Buffer
	testutils.SetMainVersion(t, "invalid")
	if buildinfo.ReleaseBuild {
		checkAndWarn("4.10.1", &buf)
		assert.Empty(t, buf.String())
	} else {
		assert.Panics(t, func() {
			checkAndWarn("4.10.1", &buf)
		})
	}
}

func TestCentralVersionClientInterceptor(t *testing.T) {
	cases := map[string]struct {
		suppress       bool
		centralVersion string
		expectWarning  string
	}{
		"incompatible version warns": {
			centralVersion: "4.2.0",
			expectWarning:  "outside the supported version skew range",
		},
		"compatible version is silent": {
			centralVersion: "4.10.6",
		},
		"incompatible version does not warn if suppressed": {
			suppress:       true,
			centralVersion: "4.2.0",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			testutils.SetMainVersion(t, "4.8.0")
			t.Cleanup(func() {
				UnsuppressVersionMismatchWarningForTesting(t)
			})

			var buf bytes.Buffer
			conn := setupServerAndClient(t,
				[]grpc.UnaryServerInterceptor{injectVersionHeaderInterceptor(t, tc.centralVersion)},
				[]grpc.UnaryClientInterceptor{CentralVersionClientInterceptor(&buf)},
			)

			client := v1.NewMetadataServiceClient(conn)
			if tc.suppress {
				SuppressVersionMismatchWarning()
			}
			_, err := client.GetMetadata(context.Background(), &v1.Empty{})
			require.NoError(t, err)

			if tc.expectWarning != "" {
				assert.Contains(t, buf.String(), tc.expectWarning)
				assert.Contains(t, buf.String(), "Compatible Centrals:")
			} else {
				assert.Empty(t, buf.String())
			}
		})
	}
}

func TestCentralVersionClientInterceptor_WithRealServerInterceptor(t *testing.T) {
	// When the client is authenticated, CentralVersionServerInterceptor serves the same version so no warning is raised.
	// When the client is not authenticated, CentralVersionServerInterceptor does not return the header and no warning is raised either.
	cases := map[string]struct {
		authenticated bool
	}{
		"authenticated with matching versions is silent": {authenticated: true},
		"anonymous is silent":                            {authenticated: false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			testutils.SetMainVersion(t, "4.8.0")
			t.Cleanup(func() {
				UnsuppressVersionMismatchWarningForTesting(t)
			})

			var serverInterceptors []grpc.UnaryServerInterceptor
			if tc.authenticated {
				serverInterceptors = append(serverInterceptors, injectIdentityInterceptor(t))
			}
			serverInterceptors = append(serverInterceptors, versionheader.CentralVersionServerInterceptor())

			var buf bytes.Buffer
			conn := setupServerAndClient(t,
				serverInterceptors,
				[]grpc.UnaryClientInterceptor{CentralVersionClientInterceptor(&buf)},
			)

			client := v1.NewMetadataServiceClient(conn)
			_, err := client.GetMetadata(context.Background(), &v1.Empty{})
			require.NoError(t, err)

			assert.Empty(t, buf.String())
		})
	}
}

func TestCentralVersionClientInterceptor_WarnsOnlyOnce(t *testing.T) {
	testutils.SetMainVersion(t, "4.8.0")
	t.Cleanup(func() {
		UnsuppressVersionMismatchWarningForTesting(t)
	})

	var buf bytes.Buffer
	conn := setupServerAndClient(t,
		[]grpc.UnaryServerInterceptor{
			injectIdentityInterceptor(t),
			injectVersionHeaderInterceptor(t, "4.2.0"),
		},
		[]grpc.UnaryClientInterceptor{CentralVersionClientInterceptor(&buf)},
	)

	client := v1.NewMetadataServiceClient(conn)
	_, err := client.GetMetadata(context.Background(), &v1.Empty{})
	require.NoError(t, err)
	assert.NotEmpty(t, buf.String())

	buf.Reset()

	_, err = client.GetMetadata(context.Background(), &v1.Empty{})
	require.NoError(t, err)
	assert.Empty(t, buf.String(), "warning should not be emitted a second time")
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
		"incompatible ahead":  {versioncompatibility.IncompatibleAhead, false, "ahead of roxctl"},
		"incompatible behind": {versioncompatibility.IncompatibleBehind, false, "behind roxctl"},
		"unknown":             {versioncompatibility.Unknown, true, ""},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			summary, _ := guidance(tt.c)
			if tt.wantEmpty {
				assert.Empty(t, summary)
			} else {
				require.NotEmpty(t, summary)
				assert.Contains(t, summary, tt.contains)
			}
		})
	}
}

func TestCheckCentralVersion(t *testing.T) {
	testutils.SetMainVersion(t, "5.0.0-testing")
	productstreams.OverrideBumpsForTesting(t, testBumpsYAML)

	tests := map[string]struct {
		centralVersion string
		wantCompat     versioncompatibility.Compatibility
		wantContains   []string
	}{
		"matched": {
			centralVersion: "5.0.2",
			wantCompat:     versioncompatibility.Matched,
			wantContains:   []string{"matched with Central"},
		},
		"compatible ahead": {
			centralVersion: "5.3.1",
			wantCompat:     versioncompatibility.CompatibleAhead,
			wantContains:   []string{"ahead of roxctl", "Use newer roxctl"},
		},
		"compatible behind": {
			centralVersion: "4.10.1",
			wantCompat:     versioncompatibility.CompatibleBehind,
			wantContains:   []string{"behind roxctl", "Central upgrade"},
		},
		"incompatible ahead": {
			centralVersion: "5.6.0",
			wantCompat:     versioncompatibility.IncompatibleAhead,
			wantContains:   []string{"ahead of roxctl", "Use newer roxctl"},
		},
		"incompatible behind": {
			centralVersion: "4.5.2",
			wantCompat:     versioncompatibility.IncompatibleBehind,
			wantContains:   []string{"behind roxctl", "Central upgrade"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(tt *testing.T) {
			res, err := CheckCentralVersion(tc.centralVersion)
			require.NoError(tt, err)

			assert.Equal(tt, tc.centralVersion, res.CentralVersion)
			assert.Equal(tt, "5.0.0-testing", res.RoxctlVersion)
			assert.Equal(tt, tc.wantCompat.String(), res.Compatibility)
			assert.Equal(tt, tc.wantCompat.DisplayName(), res.DisplayName)
			assert.NotEmpty(tt, res.CompatibleCentralVersions)
			for _, s := range tc.wantContains {
				assert.Contains(tt, res.Summary+res.Recommendation, s)
			}
		})
	}
}

// --- helpers and mocks ---

type fakeIdentity struct {
	authn.Identity
}

type fakeMetadataServer struct {
	v1.UnimplementedMetadataServiceServer
}

func (f *fakeMetadataServer) GetMetadata(_ context.Context, _ *v1.Empty) (*v1.Metadata, error) {
	return &v1.Metadata{Version: "test"}, nil
}

func setupServerAndClient(t *testing.T, serverInterceptors []grpc.UnaryServerInterceptor, clientInterceptors []grpc.UnaryClientInterceptor) *grpc.ClientConn {
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(grpc.ChainUnaryInterceptor(serverInterceptors...))
	v1.RegisterMetadataServiceServer(server, &fakeMetadataServer{})

	go func() {
		assert.NoError(t, server.Serve(listener))
	}()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(clientInterceptors...),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
		assert.NoError(t, conn.Close())
		assert.NoError(t, listener.Close())
	})
	return conn
}

func injectIdentityInterceptor(t testing.TB) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx = authn.ContextWithIdentity(ctx, fakeIdentity{}, t)
		return handler(ctx, req)
	}
}

func injectVersionHeaderInterceptor(t testing.TB, centralVersion string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		require.NoError(t, grpc.SetHeader(ctx, metadata.Pairs(clientconn.CentralVersionHeader, centralVersion)))
		return handler(ctx, req)
	}
}
