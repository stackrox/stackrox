package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	golog "log"
	"net"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/alpn"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/tlsutils"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	downgradingServer "golang.stackrox.io/grpc-http1/server"
	"google.golang.org/grpc"
)

const (
	defaultMaxHTTP2ConcurrentStreams = 100 // HTTP/2 spec recommendation for minimum value
)

var (
	maxHTTP2ConcurrentStreamsSetting = env.RegisterIntegerSetting("ROX_HTTP2_MAX_CONCURRENT_STREAMS", defaultMaxHTTP2ConcurrentStreams)
)

func maxHTTP2ConcurrentStreams() uint32 {
	if maxHTTP2ConcurrentStreamsSetting.IntegerSetting() <= 0 {
		return defaultMaxHTTP2ConcurrentStreams
	}

	return uint32(maxHTTP2ConcurrentStreamsSetting.IntegerSetting())
}

// EndpointConfig configures an endpoint through which the server is exposed.
type EndpointConfig struct {
	ListenEndpoint string
	Optional       bool
	TLS            verifier.TLSConfigurer

	ServeGRPC, ServeHTTP bool

	NoHTTP2                 bool
	DenyMisdirectedRequests bool
}

// Kind returns a human-readable description of this endpoint.
func (c *EndpointConfig) Kind() string {
	var sb strings.Builder
	if c.TLS == nil {
		sb.WriteString("Plaintext")
	} else {
		sb.WriteString("TLS-enabled")
	}
	sb.WriteRune(' ')
	if c.ServeHTTP && c.ServeGRPC {
		sb.WriteString("multiplexed HTTP/gRPC")
	} else if c.ServeHTTP {
		sb.WriteString("HTTP")
	} else if c.ServeGRPC {
		sb.WriteString("gRPC")
	} else {
		sb.WriteString("dummy")
	}
	return sb.String()
}

func tlsHandshakeErrorHandler(conn net.Conn, e error) {
	recordHdrErr, ok := e.(tls.RecordHeaderError)
	if !ok || recordHdrErr.Conn == nil {
		log.Debugf("TLS handshake error from %q: %v", conn.RemoteAddr(), e)
		return
	}

	// If the handshake failed due to the client not speaking TLS, assume they're speaking
	// plain HTTP/1 or HTTP/2 and write back either a `400` response or a `GOAWAY` frame.
	//
	// TODO(ROX-6114): Consider replying with `301` or `308`, at least for HTTP/1. For
	//   this to work, we need to get more than just the first 5 bytes of the request.
	switch string(recordHdrErr.RecordHeader[:]) {
	case "DELET", "GET /", "HEAD ", "OPTIO", "PATCH", "POST ", "PUT /", "TRACE":
		_, _ = io.WriteString(recordHdrErr.Conn,
			"HTTP/1.0 400 Bad Request\r\n\r\nClient sent an HTTP request to an HTTPS server.\n")
	case "PRI *":
		framer := http2.NewFramer(recordHdrErr.Conn, recordHdrErr.Conn)
		_ = framer.WriteSettingsAck()
		_ = framer.WriteGoAway(0, http2.ErrCodeInadequateSecurity, []byte("Client sent an HTTP request to an HTTPS server.\n"))
	}

	log.Debugf("TLS record header '%s ...' from %q is invalid: %v", recordHdrErr.RecordHeader, conn.RemoteAddr(), e)
	_ = recordHdrErr.Conn.Close()
}

func checkMisdirectedRequest(req *http.Request) error {
	if req.TLS == nil {
		return nil // we can only detect misdirected requests with TLS
	}
	if !req.ProtoAtLeast(2, 0) {
		return nil // connection caolescing requires HTTP/2.0 or higher
	}
	tlsServerName := req.TLS.ServerName
	if tlsServerName == "" {
		return nil // need an SNI ServerName
	}
	httpHostName, _, _, err := netutil.ParseEndpoint(req.Host)
	if httpHostName == "" || err != nil {
		return nil // need a valid HTTP Host or :authority header
	}
	// Host may be an IP address (or IP:port, see https://datatracker.ietf.org/doc/html/rfc7230#section-5.4), but that
	// can never be a valid ServerName, so we have nothing to compare.
	if net.ParseIP(httpHostName) != nil {
		return nil
	}
	if tlsServerName == httpHostName {
		return nil
	}
	// Whenever we have both a ServerName from SNI as well as a hostname from the :authority pseudo-header, enforce
	// that they must match. While it is possible that the server is exposed under different DNS names that are both
	// valid for the same served certificate (which would be a legitimate use case for connection coalescing), this
	// case is rare enough that no harm is done if we require separate TCP connections where one would suffice.
	return errors.Errorf("request was intended for host %s, but sent over a connection established for host %s", httpHostName, tlsServerName)
}

func denyMisdirectedRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if err := checkMisdirectedRequest(req); err != nil {
			http.Error(w, err.Error(), http.StatusMisdirectedRequest)
			return
		}
		next.ServeHTTP(w, req)
	})
}

func (c *EndpointConfig) instantiate(httpHandler http.Handler, grpcSrv *grpc.Server) (net.Addr, []serverAndListener, error) {
	lis, err := net.Listen("tcp", asEndpoint(c.ListenEndpoint))
	if err != nil {
		return nil, nil, err
	}

	var httpLis, grpcLis net.Listener

	var result []serverAndListener

	var tlsConf *tls.Config
	if c.TLS != nil {
		tlsConf, err = c.TLS.TLSConfig()
		if err != nil {
			return nil, nil, errors.Wrap(err, "configuring TLS")
		}
	}

	if c.NoHTTP2 && tlsConf != nil {
		tlsConf = tlsConf.Clone()
		tlsConf.NextProtos = sliceutils.Without(tlsConf.NextProtos, []string{"h2", alpn.PureGRPCALPNString})
	}

	if tlsConf != nil {
		if c.ServeGRPC && !c.NoHTTP2 {
			tlsConf = alpn.ApplyPureGRPCALPNConfig(tlsConf)
		}
		lis = tls.NewListener(lis, tlsConf)

		if c.ServeGRPC && c.ServeHTTP {
			protoMap := map[string]*net.Listener{
				alpn.PureGRPCALPNString: &grpcLis,
				"":                      &httpLis,
			}

			tlsutils.ALPNDemux(lis, protoMap, tlsutils.ALPNDemuxConfig{OnHandshakeError: tlsHandshakeErrorHandler})
		}
	}

	// Default to listen on the main listener, HTTP first
	if c.ServeHTTP && httpLis == nil {
		httpLis = lis
	} else if c.ServeGRPC && grpcLis == nil {
		grpcLis = lis
	}

	if httpLis != nil {
		actualHTTPHandler := httpHandler
		if c.ServeGRPC {
			actualHTTPHandler = downgradingServer.CreateDowngradingHandler(grpcSrv, actualHTTPHandler, downgradingServer.PreferGRPCWeb(true))
		}

		httpSrv := &http.Server{
			Handler:   actualHTTPHandler,
			TLSConfig: tlsConf,
			ErrorLog:  golog.New(httpErrorLogger{}, "", golog.LstdFlags),
		}
		if !c.NoHTTP2 {
			h2Srv := http2.Server{
				MaxConcurrentStreams: maxHTTP2ConcurrentStreams(),
			}
			if err := http2.ConfigureServer(httpSrv, &h2Srv); err != nil {
				log.Warnf("Failed to instantiate endpoint listening at %q for HTTP/2", c.ListenEndpoint)
			} else {
				httpSrv.Handler = h2c.NewHandler(actualHTTPHandler, &h2Srv)
			}
			if c.DenyMisdirectedRequests && tlsConf != nil {
				// When using HTTP/2 over TLS, connection coalescing in conjunction with wildcard or multi-SAN
				// certificates may cause this server to receive requests not intended for it. If
				// DenyMisdirectedRequests is set to true, deny such requests outright with a 421 (Misdirected Request)
				// status code.
				httpSrv.Handler = denyMisdirectedRequest(httpSrv.Handler)
			}
		}
		result = append(result, serverAndListener{
			srv:      httpSrv,
			listener: httpLis,
			endpoint: c,
			stopper: func() {
				if err := httpSrv.Shutdown(context.Background()); err != nil {
					log.Warnf("Stopping HTTP listener: %s", err)
				}
			},
		})
	}
	if grpcLis != nil {
		result = append(result, serverAndListener{
			srv:      grpcSrv,
			listener: grpcLis,
			endpoint: c,
		})
	}

	return lis.Addr(), result, nil
}

// asEndpoint returns an all-interface endpoint of form `:<port>` if `portOrEndpoint` is a port only (does not contain
// a ':'). Otherwise, `portOrEndpoint` is returned as-is.
func asEndpoint(portOrEndpoint string) string {
	if !strings.ContainsRune(portOrEndpoint, ':') {
		return fmt.Sprintf(":%s", portOrEndpoint)
	}
	return portOrEndpoint
}
