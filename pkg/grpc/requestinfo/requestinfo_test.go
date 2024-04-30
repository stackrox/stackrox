package requestinfo

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/netutil/pipeconn"
	"github.com/stackrox/rox/pkg/testutils"

	pb "github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var insecureTLSConfig = &tls.Config{InsecureSkipVerify: true}
var insecureSkipVerify = credentials.NewTLS(insecureTLSConfig)

func Test_PureHTTP(t *testing.T) {
	handler := NewRequestInfoHandler()

	var receivedRequest *http.Request
	var receivedRI *RequestInfo

	server := httptest.NewServer(handler.HTTPIntercept(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedRequest = r
		ri := FromContext(r.Context())
		receivedRI = &ri
	})))
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL+"/v1/test", nil)
	req.Header.Add("test-key", "test value")

	client := http.Client{}
	_, err := client.Do(req)
	require.NoError(t, err)

	require.NotNil(t, receivedRequest)
	assert.Equal(t, req.Method, receivedRequest.Method)
	assert.Equal(t, req.Header.Get("test-key"), receivedRequest.Header.Get("test-key"))
	assert.Equal(t, "Go-http-client/1.1", receivedRequest.Header.Get("user-agent"))

	require.NotNil(t, receivedRI)
	assert.Equal(t, receivedRequest.Header, receivedRI.HTTPRequest.Headers)
	assert.NotNil(t, receivedRI.Metadata)
	assert.Equal(t, []string{"test value"}, receivedRI.Metadata.Get("test-key"))
}

type pingService struct {
	pb.UnimplementedPingServiceServer
	receivedRI *RequestInfo
}

func (s *pingService) Ping(ctx context.Context, in *pb.Empty) (*pb.PongMessage, error) {
	ri := FromContext(ctx)
	s.receivedRI = &ri
	return &pb.PongMessage{Status: "pong"}, nil
}

func Test_PureGRPC(t *testing.T) {
	handler := NewRequestInfoHandler()

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(
		contextutil.UnaryServerInterceptor(handler.UpdateContextForGRPC)))

	serviceInstance := &pingService{}
	pb.RegisterPingServiceServer(grpcServer, serviceInstance)

	server := httptest.NewUnstartedServer(http.HandlerFunc(grpcServer.ServeHTTP))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	conn, err := grpc.Dial(server.Listener.Addr().String(),
		grpc.WithTransportCredentials(insecureSkipVerify))
	require.NoError(t, err)
	defer conn.Close()

	c := pb.NewPingServiceClient(conn)
	resp, err := c.Ping(context.Background(), &pb.Empty{})
	require.NoError(t, err)
	require.Equal(t, "pong", resp.Status)

	receivedRI := serviceInstance.receivedRI
	require.NotNil(t, receivedRI)
	require.Nil(t, receivedRI.HTTPRequest)
	assert.NotNil(t, receivedRI.Metadata)
	assert.Equal(t, []string{"application/grpc"}, receivedRI.Metadata.Get("content-type"))
	assert.Contains(t, receivedRI.Metadata.Get("user-agent")[0], "grpc-go/")
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

	err := pb.RegisterPingServiceHandlerFromEndpoint(ctx, gateway, lis.Addr().String(),
		[]grpc.DialOption{
			grpc.WithTransportCredentials(creds),
			grpc.WithContextDialer(func(ctx context.Context, url string) (net.Conn, error) {
				return dialContext(ctx)
			}),
			grpc.WithUserAgent("gateway agent"),
		})
	require.NoError(t, err)

	server := httptest.NewUnstartedServer(gateway)
	server.TLS = insecureTLSConfig
	server.StartTLS()
	defer server.Close()

	client := http.Client{Transport: &http.Transport{
		TLSClientConfig:   insecureTLSConfig,
		ForceAttemptHTTP2: true,
	}}

	req, _ := http.NewRequest("GET", server.URL+"/v1/ping", nil)
	req.Header.Add("user-agent", "test agent")
	req.Header.Add("test-key", "test value")
	_, err = client.Do(req)
	require.NoError(t, err)

	receivedRI := serviceInstance.receivedRI
	require.NotNil(t, receivedRI)

	require.NotNil(t, receivedRI.HTTPRequest)
	assert.Equal(t, "test agent", receivedRI.HTTPRequest.Headers.Get("user-agent"))
	assert.Equal(t, "test value", receivedRI.HTTPRequest.Headers.Get("test-key"))

	assert.NotNil(t, receivedRI.Metadata)
	assert.Equal(t, []string{"application/grpc"}, receivedRI.Metadata.Get("content-type"))
	assert.Contains(t, receivedRI.Metadata.Get("user-agent")[0], "gateway agent")
	prefixedUserAgentKey, _ := runtime.DefaultHeaderMatcher("user-agent")
	assert.Contains(t, receivedRI.Metadata.Get(prefixedUserAgentKey)[0], "test agent")
}
