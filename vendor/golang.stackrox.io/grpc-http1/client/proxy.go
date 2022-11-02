// Copyright (c) 2020 StackRox Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License

package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.stackrox.io/grpc-http1/internal/grpcproto"
	"golang.stackrox.io/grpc-http1/internal/grpcweb"
	"golang.stackrox.io/grpc-http1/internal/httputils"
	"golang.stackrox.io/grpc-http1/internal/pipeconn"
	"golang.stackrox.io/grpc-http1/internal/stringutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
)

func modifyResponse(resp *http.Response) error {
	// Check if the response is an error response right away, and attempt to display a more useful
	// message than gRPC does by default. We still delegate to the default gRPC behavior for 200 responses
	// which are otherwise invalid.
	if err := httputils.ExtractResponseError(resp); err != nil {
		return errors.Wrap(err, "receiving gRPC response from remote endpoint")
	}

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

// Fake a gRPC status with the given transport error
func writeError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/grpc")
	w.Header().Add("Trailer", "Grpc-Status")
	w.Header().Add("Trailer", "Grpc-Message")
	w.WriteHeader(http.StatusOK)

	w.Header().Set("Grpc-Status", fmt.Sprintf("%d", codes.Unavailable))
	errMsg := errors.Wrap(err, "transport").Error()
	w.Header().Set("Grpc-Message", grpcproto.EncodeGrpcMessage(errMsg))
}

func createReverseProxy(endpoint string, transport http.RoundTripper, insecure, forceDowngrade bool) *httputil.ReverseProxy {
	scheme := "https"
	if insecure {
		scheme = "http"
	}
	return &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			if forceDowngrade {
				req.ProtoMajor, req.ProtoMinor, req.Proto = 1, 1, "HTTP/1.1"
				req.Header.Del("TE")
				req.Header.Del("Accept")
				req.Header.Add(grpcweb.GRPCWebOnlyHeader, "true")
			} else {
				req.Header.Add("Accept", "application/grpc")
			}
			req.Header.Add("Accept", "application/grpc-web")
			req.URL.Scheme = scheme
			req.URL.Host = endpoint
		},
		Transport:      transport,
		ModifyResponse: modifyResponse,
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			writeError(w, err)
		},
		// No need to set FlushInterval, as we force the writer to operate in unbuffered mode/flushing after every
		// write.
	}
}

func createTransport(tlsClientConf *tls.Config, forceHTTP2 bool, extraH2ALPNs []string) (http.RoundTripper, error) {
	if forceHTTP2 {
		transport := &http2.Transport{
			AllowHTTP:       true,
			TLSClientConfig: tlsClientConf,
		}
		if tlsClientConf == nil {
			transport.DialTLS = func(network, addr string, _ *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			}
		}
		return transport, nil
	}

	transport := &http.Transport{
		ForceAttemptHTTP2: true,
	}

	if tlsClientConf != nil {
		transport.TLSClientConfig = tlsClientConf.Clone()
	}
	if err := http2.ConfigureTransport(transport); err != nil {
		return nil, errors.Wrap(err, "configuring transport for HTTP/2 use")
	}

	// Make sure the transport for any extra HTTP/2-like ALPN string behaves like for HTTP/2.
	for _, extraALPN := range extraH2ALPNs {
		transport.TLSNextProto[extraALPN] = transport.TLSNextProto["h2"]
	}

	return transport, nil
}

func createClientProxy(endpoint string, tlsClientConf *tls.Config, forceHTTP2, forceDowngrade bool, extraH2ALPNs []string) (*http.Server, pipeconn.DialContextFunc, error) {
	transport, err := createTransport(tlsClientConf, forceHTTP2, extraH2ALPNs)
	if err != nil {
		return nil, nil, errors.Wrap(err, "creating transport")
	}
	proxy := createReverseProxy(endpoint, transport, tlsClientConf == nil, forceDowngrade)
	return makeProxyServer(proxy)
}

// ConnectViaProxy establishes a gRPC client connection via a HTTP/2 proxy that handles endpoints behind HTTP/1.x proxies.
// Use the WithWebSocket() ConnectOption if you want to connect to a server via WebSocket.
// Otherwise, setting it to false will use a gRPC-Web "downgrade", as needed.
//
// Using WebSocket will allow for both streaming and non-streaming gRPC requests, but is not adaptive.
// Using gRPC-Web "downgrades" will only allow for non-streaming gRPC requests, but will only downgrade if necessary.
// This method supports server-streaming requests, but only if there isn't a proxy in the middle that buffers chunked responses.
func ConnectViaProxy(ctx context.Context, endpoint string, tlsClientConf *tls.Config, opts ...ConnectOption) (*grpc.ClientConn, error) {
	var connectOpts connectOptions
	for _, opt := range opts {
		opt.apply(&connectOpts)
	}

	var proxy *http.Server
	var dialCtx pipeconn.DialContextFunc
	var err error

	if connectOpts.useWebSocket {
		proxy, dialCtx, err = createClientWSProxy(endpoint, tlsClientConf)
	} else {
		proxy, dialCtx, err = createClientProxy(endpoint, tlsClientConf, connectOpts.forceHTTP2, connectOpts.forceDowngrade, connectOpts.extraH2ALPNs)
	}

	if err != nil {
		return nil, errors.Wrap(err, "creating client proxy")
	}

	return dialGRPCServer(ctx, proxy, makeDialOpts(endpoint, dialCtx, tlsClientConf, connectOpts))
}

func makeProxyServer(handler http.Handler) (*http.Server, pipeconn.DialContextFunc, error) {
	lis, dialCtx := pipeconn.NewPipeListener()

	var http2Srv http2.Server
	srv := &http.Server{
		Addr:    lis.Addr().String(),
		Handler: h2c.NewHandler(nonBufferingHandler(handler), &http2Srv),
	}
	if err := http2.ConfigureServer(srv, &http2Srv); err != nil {
		return nil, nil, errors.Wrap(err, "configuring HTTP/2 server")
	}

	go func() {
		if err := srv.Serve(lis); err != nil && err != http.ErrServerClosed {
			glog.Warningf("Unexpected error returned from serving gRPC proxy server: %v", err)
		}
	}()

	return srv, dialCtx, nil
}

func makeDialOpts(endpoint string, dialCtx pipeconn.DialContextFunc, tlsClientConf *tls.Config, connectOpts connectOptions) []grpc.DialOption {
	dialOpts := make([]grpc.DialOption, 0, len(connectOpts.dialOpts)+2)
	dialOpts = append(dialOpts, grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
		return dialCtx(ctx)
	}))
	if tlsClientConf != nil {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(newCredsFromSideChannel(endpoint, credentials.NewTLS(tlsClientConf))))
	}
	dialOpts = append(dialOpts, connectOpts.dialOpts...)

	return dialOpts
}

func dialGRPCServer(ctx context.Context, proxy *http.Server, dialOpts []grpc.DialOption) (*grpc.ClientConn, error) {
	cc, err := grpc.DialContext(ctx, proxy.Addr, dialOpts...)
	if err != nil {
		_ = proxy.Close()
		return nil, err
	}
	go closeServerOnConnShutdown(proxy, cc)
	return cc, nil
}

func closeServerOnConnShutdown(srv *http.Server, cc *grpc.ClientConn) {
	for state := cc.GetState(); state != connectivity.Shutdown; state = cc.GetState() {
		cc.WaitForStateChange(context.Background(), state)
	}
	if err := srv.Close(); err != nil {
		glog.Warningf("Error closing gRPC proxy server: %v", err)
	}
}
