package grpc

import (
	"crypto/tls"
	"fmt"
	"io"
	golog "log"
	"net"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/grpc/alpn"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/tlsutils"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	downgradingServer "golang.stackrox.io/grpc-http1/server"
	"google.golang.org/grpc"
)

// EndpointConfig configures an endpoint through which the server is exposed.
type EndpointConfig struct {
	ListenEndpoint string
	Optional       bool
	TLS            verifier.TLSConfigurer

	ServeGRPC, ServeHTTP bool

	NoHTTP2 bool
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

func isMisdirectedRequest(req *http.Request) bool {
	if req.TLS == nil {
		return false // we can only detect misdirected requests with TLS
	}
	if !req.ProtoAtLeast(2, 0) {
		return false // connection caolescing requires HTTP/2.0 or higher
	}
	tlsServerName := req.TLS.ServerName
	if tlsServerName == "" {
		return false // need an SNI ServerName
	}
	httpHostName := stringutils.GetUpTo(req.Host, ":") // may be of form host:port
	if httpHostName == "" {
		return false // need a HTTP Host or :authority header
	}
	if tlsServerName == httpHostName {
		return false
	}
	// Even if hostname and server name are distinct, let's limit ourselves to only classifying requests
	// as positively misdirected if the discrepancy can be explained by a wildcard cert.
	tlsServerNameDomain := stringutils.GetAfter(tlsServerName, ".")
	if tlsServerNameDomain == "" {
		return false
	}
	httpHostNameDomain := stringutils.GetAfter(httpHostName, ".")
	if httpHostNameDomain == "" {
		return false
	}
	return tlsServerNameDomain == httpHostNameDomain
}

func denyMisdirectedRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if isMisdirectedRequest(req) {
			http.Error(w, "received connection for unexpected hostname", http.StatusMisdirectedRequest)
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

	if c.NoHTTP2 {
		tlsConf = tlsConf.Clone()
		tlsConf.NextProtos = sliceutils.StringDifference(tlsConf.NextProtos, []string{"h2", alpn.PureGRPCALPNString})
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
			var h2Srv http2.Server
			if err := http2.ConfigureServer(httpSrv, &h2Srv); err != nil {
				log.Warnf("Failed to instantiate endpoint listening at %q for HTTP/2", c.ListenEndpoint)
			} else {
				httpSrv.Handler = h2c.NewHandler(actualHTTPHandler, &h2Srv)
			}
			httpSrv.Handler = denyMisdirectedRequest(httpSrv.Handler)
		}
		result = append(result, serverAndListener{
			srv:      httpSrv,
			listener: httpLis,
			endpoint: c,
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
