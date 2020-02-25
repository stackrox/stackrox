package client

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httputil"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/grpc/alpn"
	"github.com/stackrox/rox/pkg/grpcweb"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/netutil/pipeconn"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/stringutils"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
)

var (
	log = logging.LoggerForModule()
)

func modifyResponse(resp *http.Response) error {
	if resp.ContentLength == 0 {
		// Make sure headers do not get flushed, as otherwise the gRPC client will complain about missing trailers.
		resp.Header.Set(dontFlushHeadersHeaderKey, "true")
	}
	contentType, contentSubType := stringutils.Split2(resp.Header.Get("Content-Type"), "+")
	if contentType != "application/grpc-web" {
		// No modification necessary if we aren't handling a gRPC web response.
		return nil
	}

	respCT := "application/grpc"
	if contentSubType != "" {
		respCT += "+" + contentSubType
	}
	resp.Header.Set("Content-Type", respCT)

	if resp.Body != nil {
		resp.Body = grpcweb.NewResponseReader(resp.Body, &resp.Trailer, nil)
	}
	return nil
}

func createReverseProxy(endpoint string, transport http.RoundTripper, insecure bool) *httputil.ReverseProxy {
	scheme := "https"
	if insecure {
		scheme = "http"
	}
	return &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.Header.Add("Accept", "application/grpc")
			req.Header.Add("Accept", "application/grpc-web")
			req.URL.Scheme = scheme
			req.URL.Host = endpoint
		},
		Transport:      transport,
		ModifyResponse: modifyResponse,
		// No need to set FlushInterval, as we force the writer to operate in unbuffered mode/flushing after every
		// write.
	}
}

func createTransport(tlsClientConf *tls.Config) (http.RoundTripper, error) {
	transport := &http.Transport{
		ForceAttemptHTTP2: true,
	}

	if tlsClientConf != nil {
		clientConfForTransport := tlsClientConf.Clone()
		nextProtos := sliceutils.ConcatStringSlices(clientconn.NextProtos, tlsClientConf.NextProtos, []string{"http/1.1", "http/1.0"})
		clientConfForTransport.NextProtos = sliceutils.StringUnique(nextProtos)
		transport.TLSClientConfig = clientConfForTransport
	}
	if err := http2.ConfigureTransport(transport); err != nil {
		return nil, errors.Wrap(err, "configuring transport for HTTP/2 use")
	}

	// Make sure the transport for our "pure gRPC" custom application-level protocol behaves like for HTTP/2.
	transport.TLSNextProto[alpn.PureGRPCALPNString] = transport.TLSNextProto["h2"]

	return transport, nil
}

func createClientProxy(endpoint string, tlsClientConf *tls.Config) (*http.Server, pipeconn.DialContextFunc, error) {
	transport, err := createTransport(tlsClientConf)
	if err != nil {
		return nil, nil, errors.Wrap(err, "creating transport")
	}
	proxy := createReverseProxy(endpoint, transport, tlsClientConf == nil)

	var http2srv http2.Server
	srv := &http.Server{
		Handler: h2c.NewHandler(nonBufferingHandler(proxy), &http2srv),
	}
	if err := http2.ConfigureServer(srv, &http2srv); err != nil {
		return nil, nil, errors.Wrap(err, "configuring HTTP/2 server")
	}

	listener, dialCtx := pipeconn.NewPipeListener()

	srv.Addr = listener.Addr().String()
	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Warnf("Unexpected error returned from serving gRPC proxy server: %v", err)
		}
	}()

	return srv, dialCtx, nil
}

// ConnectViaProxy establishes a gRPC client connection via a HTTP/2 proxy that handles endpoints behind HTTP/1 proxies.
func ConnectViaProxy(ctx context.Context, endpoint string, tlsClientConf *tls.Config, extraOpts ...grpc.DialOption) (*grpc.ClientConn, error) {
	proxySrv, dialCtx, err := createClientProxy(endpoint, tlsClientConf)
	if err != nil {
		return nil, errors.Wrap(err, "creating client proxy")
	}

	dialOpts := make([]grpc.DialOption, 0, len(extraOpts)+2)
	dialOpts = append(dialOpts, grpc.WithContextDialer(func(ctx context.Context, target string) (net.Conn, error) {
		return dialCtx(ctx)
	}))
	if tlsClientConf != nil {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(newCredsFromSideChannel(endpoint, credentials.NewTLS(tlsClientConf))))
	}
	dialOpts = append(dialOpts, extraOpts...)

	cc, err := grpc.DialContext(ctx, proxySrv.Addr, dialOpts...)
	if err != nil {
		_ = proxySrv.Close()
		return nil, err
	}
	go closeServerOnConnShutdown(proxySrv, cc)
	return cc, nil
}

func closeServerOnConnShutdown(srv *http.Server, cc *grpc.ClientConn) {
	for state := cc.GetState(); state != connectivity.Shutdown; state = cc.GetState() {
		cc.WaitForStateChange(context.Background(), state)
	}
	if err := srv.Close(); err != nil {
		log.Warnf("Error closing gRPC proxy server: %v", err)
	}
}
