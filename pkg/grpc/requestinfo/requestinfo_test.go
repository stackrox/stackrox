package requestinfo

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	metautils "github.com/grpc-ecosystem/go-grpc-middleware/v2/metadata"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	pb "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/netutil/pipeconn"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

var insecureTLSConfig = &tls.Config{InsecureSkipVerify: true}
var insecureSkipVerify = credentials.NewTLS(insecureTLSConfig)

const userAgentKey = "User-Agent"

func makeTestRequest(url string) *http.Request {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("test-key", "test value")
	req.Header.Add(userAgentKey, "test agent")
	return req
}

func testRequest(t *testing.T, req *HTTPRequest) {
	require.NotNil(t, req)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "test value", req.Headers.Get("test-key"))
	assert.Equal(t, "test agent", req.Headers.Get(userAgentKey))
}

type pingService struct {
	pb.UnimplementedPingServiceServer
	ri *RequestInfo
}

func (s *pingService) Ping(ctx context.Context, in *pb.Empty) (*pb.PongMessage, error) {
	ri := FromContext(ctx)
	s.ri = &ri
	pm := &pb.PongMessage{}
	pm.SetStatus("pong")
	return pm, nil
}

func Test_PureGRPC(t *testing.T) {
	handler := NewRequestInfoHandler()

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(
		contextutil.UnaryServerInterceptor(handler.UpdateContextForGRPC)))

	serviceInstance := &pingService{}
	pb.RegisterPingServiceServer(grpcServer, serviceInstance)

	server := httptest.NewUnstartedServer(handler.HTTPIntercept(http.HandlerFunc(grpcServer.ServeHTTP)))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	conn, err := grpc.Dial(server.Listener.Addr().String(),
		grpc.WithTransportCredentials(insecureSkipVerify))
	require.NoError(t, err)
	defer utils.IgnoreError(conn.Close)

	c := pb.NewPingServiceClient(conn)
	resp, err := c.Ping(context.Background(), &pb.Empty{})
	require.NoError(t, err)
	require.Equal(t, "pong", resp.GetStatus())

	ri := serviceInstance.ri
	require.NotNil(t, ri)
	require.Nil(t, ri.HTTPRequest)
	assert.NotNil(t, ri.Metadata)
	assert.Equal(t, []string{"application/grpc"}, ri.Metadata.Get("content-type"))
	assert.Contains(t, ri.Metadata.Get(userAgentKey)[0], "grpc-go/")
}

func Test_gRPCGateway(t *testing.T) {
	handler := NewRequestInfoHandler()
	serviceInstance := &pingService{}

	cert := testutils.IssueSelfSignedCert(t, "*.example.com", "*.example.com")
	creds := credentials.NewTLS(&tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	})

	grpcServer := grpc.NewServer(grpc.Creds(creds),
		grpc.UnaryInterceptor(contextutil.UnaryServerInterceptor(
			handler.UpdateContextForGRPC)))
	pb.RegisterPingServiceServer(grpcServer, serviceInstance)

	lis, dialContext := pipeconn.NewPipeListener()
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal(err)
		}
	}()
	defer grpcServer.Stop()

	gateway := runtime.NewServeMux(runtime.WithMetadata(handler.AnnotateMD))

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	err := pb.RegisterPingServiceHandlerFromEndpoint(ctx, gateway, fmt.Sprintf("passthrough://%s", lis.Addr().String()),
		[]grpc.DialOption{
			grpc.WithTransportCredentials(creds),
			grpc.WithContextDialer(func(ctx context.Context, url string) (net.Conn, error) {
				return dialContext(ctx)
			}),
			grpc.WithUserAgent("gateway agent"),
		})
	require.NoError(t, err)

	// No handler.HTTPIntercept handler for the gateway.
	server := httptest.NewUnstartedServer(gateway)
	server.TLS = insecureTLSConfig
	server.StartTLS()
	defer server.Close()

	req := makeTestRequest(server.URL + "/v1/ping")

	client := http.Client{Transport: &http.Transport{
		TLSClientConfig:   insecureTLSConfig,
		ForceAttemptHTTP2: true,
	}}
	_, err = client.Do(req)
	require.NoError(t, err)

	receivedRI := serviceInstance.ri
	require.NotNil(t, receivedRI)

	require.NotNil(t, receivedRI.HTTPRequest)
	assert.Equal(t, "test agent", receivedRI.HTTPRequest.Headers.Get(userAgentKey))
	assert.Equal(t, "test value", receivedRI.HTTPRequest.Headers.Get("test-key"))

	assert.NotNil(t, receivedRI.Metadata)
	assert.Equal(t, []string{"application/grpc"}, receivedRI.Metadata.Get("content-type"))
	assert.Contains(t, receivedRI.Metadata.Get(userAgentKey)[0], "gateway agent")
	prefixedUserAgentKey, _ := runtime.DefaultHeaderMatcher(userAgentKey)
	assert.Contains(t, receivedRI.Metadata.Get(prefixedUserAgentKey)[0], "test agent")
}

func Test_Conversions(t *testing.T) {

	req := makeTestRequest("https://endpoint/v1/ping")

	t.Run("http flow", func(t *testing.T) {
		handler := NewRequestInfoHandler()
		var riptr *RequestInfo
		handler.HTTPIntercept(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ri := FromContext(r.Context())
			riptr = &ri
		})).ServeHTTP(nil, req)

		require.NotNil(t, riptr)
		testRequest(t, riptr.HTTPRequest)
		assert.NotNil(t, riptr.Metadata)
		// A copy of the request headers:
		assert.Equal(t, []string{"test agent"}, riptr.Metadata.Get(userAgentKey))
		assert.Equal(t, []string{"test value"}, riptr.Metadata.Get("test-key"))
	})

	t.Run("grpc flow", func(t *testing.T) {
		handler := NewRequestInfoHandler()
		ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: &net.UnixAddr{Net: "pipe"}})

		ctx = metautils.MD{}.Add("test-key", "test value").ToIncoming(ctx)

		ctx, err := handler.UpdateContextForGRPC(ctx)
		require.NoError(t, err)

		ri := FromContext(ctx)
		assert.Empty(t, ri.Metadata.Get(requestInfoMDKey))
		assert.Empty(t, ri.Metadata.Get(userAgentKey))
		assert.Equal(t, []string{"test value"}, ri.Metadata.Get("test-key"))
		require.Nil(t, ri.HTTPRequest)
	})

	t.Run("grpc-gateway flow", func(t *testing.T) {
		handler := NewRequestInfoHandler()
		ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: &net.UnixAddr{Net: "pipe"}})

		md := handler.AnnotateMD(ctx, req)
		assert.NotEmpty(t, md.Get(requestInfoMDKey))
		md.Set(userAgentKey, "gateway")

		ctx = metautils.MD(md).ToIncoming(ctx)
		ctx, err := handler.UpdateContextForGRPC(ctx)
		require.NoError(t, err)

		ri := FromContext(ctx)
		require.NotNil(t, ri.Metadata)
		assert.Empty(t, ri.Metadata.Get(requestInfoMDKey))
		assert.Equal(t, []string{"gateway"}, ri.Metadata.Get(userAgentKey))
		testRequest(t, ri.HTTPRequest)
	})
}
