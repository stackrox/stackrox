package versionheader

import (
	"context"
	"net"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

func TestUnaryServerInterceptor(t *testing.T) {
	cases := map[string]struct {
		authenticated bool
		expectHeader  bool
	}{
		"authenticated request sets version header": {
			authenticated: true,
			expectHeader:  true,
		},
		"anonymous request does not set version header": {
			authenticated: false,
			expectHeader:  false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			testutils.SetMainVersion(t, "4.8.0")

			var interceptors []grpc.UnaryServerInterceptor
			if tc.authenticated {
				interceptors = append(interceptors, injectIdentityInterceptor(t))
			}
			interceptors = append(interceptors, UnaryServerInterceptor())

			conn := setupServer(t, interceptors...)

			var md metadata.MD
			client := v1.NewMetadataServiceClient(conn)
			_, err := client.GetMetadata(context.Background(), &v1.Empty{}, grpc.Header(&md))
			require.NoError(t, err)

			vals := md.Get(clientconn.CentralVersionHeader)
			if tc.expectHeader {
				require.Len(t, vals, 1)
				assert.Equal(t, "4.8.0", vals[0])
			} else {
				assert.Empty(t, vals)
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

func injectIdentityInterceptor(t testing.TB) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx = authn.ContextWithIdentity(ctx, fakeIdentity{}, t)
		return handler(ctx, req)
	}
}

func setupServer(t *testing.T, interceptors ...grpc.UnaryServerInterceptor) *grpc.ClientConn {
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(grpc.ChainUnaryInterceptor(interceptors...))
	v1.RegisterMetadataServiceServer(server, &fakeMetadataServer{})

	go func() {
		assert.NoError(t, server.Serve(listener))
	}()

	conn, err := grpc.DialContext(context.Background(), "",
		grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	t.Cleanup(server.Stop)
	return conn
}
