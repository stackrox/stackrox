package versioncheck

import (
	"bytes"
	"context"
	"net"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/versionheader"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

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
			expectWarning:  "too new",
		},
		"incompatible ahead": {
			localVersion:   "4.8.0",
			centralVersion: "4.15.0",
			expectWarning:  "too old",
		},
		"invalid central version": {
			localVersion:   "4.8.0",
			centralVersion: "invalid",
		},
		"invalid local version": {
			localVersion:   "invalid",
			centralVersion: "4.10.1",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			testutils.SetMainVersion(t, tc.localVersion)

			var buf bytes.Buffer
			result := checkAndWarn(tc.centralVersion, &buf)

			assert.Equal(t, tc.expectWarning != "", result, "return value mismatch")
			if tc.expectWarning != "" {
				assert.Contains(t, buf.String(), tc.expectWarning)
				assert.Contains(t, buf.String(), "Compatible Centrals:")
			} else {
				assert.Empty(t, buf.String())
			}
		})
	}
}

func TestCentralVersionClientInterceptor(t *testing.T) {
	cases := map[string]struct {
		authenticated  bool
		centralVersion string
		expectWarning  string
	}{
		"incompatible version warns": {
			authenticated:  true,
			centralVersion: "4.2.0",
			expectWarning:  "too new",
		},
		"compatible version is silent": {
			authenticated:  true,
			centralVersion: "",
		},
		"anonymous request is silent": {
			authenticated:  false,
			centralVersion: "",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			testutils.SetMainVersion(t, "4.8.0")

			var serverInterceptors []grpc.UnaryServerInterceptor
			if tc.authenticated {
				serverInterceptors = append(serverInterceptors, injectIdentityInterceptor(t))
			}
			if tc.centralVersion != "" {
				serverInterceptors = append(serverInterceptors, injectVersionHeaderInterceptor(t, tc.centralVersion))
			} else {
				// Use the real interceptor: it sets the header to GetMainVersion() (same as local),
				// so versions match and no warning fires.
				serverInterceptors = append(serverInterceptors, versionheader.CentralVersionServerInterceptor())
			}

			var buf bytes.Buffer
			conn := setupServerAndClient(t,
				serverInterceptors,
				[]grpc.UnaryClientInterceptor{CentralVersionClientInterceptor(&buf)},
			)

			client := v1.NewMetadataServiceClient(conn)
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

func TestCentralVersionClientInterceptor_WarnsOnlyOnce(t *testing.T) {
	testutils.SetMainVersion(t, "4.8.0")

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
	t.Cleanup(server.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(clientInterceptors...),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
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
