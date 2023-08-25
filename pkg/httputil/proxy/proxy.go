package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	proxyReloadInterval = 5 * time.Second
)

var (
	log = logging.LoggerForModule()

	defaultProxyConfig = (&proxyConfig{}).Compile(initialEnvCfg)
	globalProxyConfig  atomic.Value

	proxyTransport *http.Transport
)

func init() {
	proxyTransport = copyDefaultTransport()
	proxyTransport.Proxy = TransportFunc
}

func getGlobalProxyConfig() *compiledConfig {
	cc, _ := globalProxyConfig.Load().(*compiledConfig)
	if cc == nil {
		return defaultProxyConfig
	}
	return cc
}

// UseWithDefaultTransport configures the default HTTP transport to use the proxy function defined in this package.
// It should be called from an `init()` function to avoid any concurrent access to fields of `http.DefaultTransport`.
func UseWithDefaultTransport() bool {
	defaultTrans, _ := http.DefaultTransport.(*http.Transport)
	if defaultTrans == nil {
		return false
	}
	defaultTrans.Proxy = TransportFunc
	return true
}

// FromConfig returns an function suitable for use as a Proxy field in an *http.Transport instance that will always
// use the latest configured proxy setting.
func FromConfig() func(*http.Request) (*url.URL, error) {
	return getGlobalProxyConfig().ProxyURL
}

// TransportFunc is a function that is suitable to use in http.Transport.ProxyFunc
func TransportFunc(req *http.Request) (*url.URL, error) {
	return FromConfig()(req)
}

// Without is a ProxyFunc for http.Transport that will always attempt a direct connection.
func Without() http.RoundTripper {
	transport := copyDefaultTransport()
	transport.Proxy = nil
	return transport
}

// RoundTripper returns something very similar to http.DefaultTransport, but with the Proxy setting changed to use
// the configuration supported by this package.
func RoundTripper() http.RoundTripper {
	return proxyTransport
}

// RoundTripperWithTLSConfig returns a round tripper like RoundTripper(), but using a custom TLS config.
func RoundTripperWithTLSConfig(tlsConf *tls.Config) http.RoundTripper {
	trans := proxyTransport.Clone()
	trans.TLSClientConfig = tlsConf
	return trans
}

// AwareDialContext implements a TCP "DialContext", but respecting the proxy configuration.
func AwareDialContext(ctx context.Context, address string) (net.Conn, error) {
	configurator := FromConfig()
	if configurator == nil {
		return defaultDialer.DialContext(ctx, "tcp", address)
	}

	fakeHTTPReq, err := http.NewRequest(http.MethodGet, fmt.Sprintf("tcp://%s", address), nil)
	if err != nil {
		return nil, utils.ShouldErr(errors.Wrapf(err, "failed to instantiate fake HTTP request for address %q", address))
	}
	proxyURL, err := configurator(fakeHTTPReq)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to determine proxy URL for address %q", address)
	}
	return DialWithProxy(ctx, proxyURL, address)
}

// AwareDialContextTLS is a convenience wrapper around AwareDialContext that establishes a TLS connection.
// It is up to the client to close the connection once it is no longer needed.
func AwareDialContextTLS(ctx context.Context, address string, tlsClientConf *tls.Config) (net.Conn, error) {
	host, _, _, err := netutil.ParseEndpoint(address)
	if err != nil {
		return nil, errors.Wrapf(err, "unparseable address %q", address)
	}

	conn, err := AwareDialContext(ctx, address)
	if err != nil {
		return nil, err
	}
	if tlsClientConf == nil {
		tlsClientConf = &tls.Config{
			ServerName: host,
		}
	} else if tlsClientConf.ServerName == "" {
		tlsClientConf = tlsClientConf.Clone()
		tlsClientConf.ServerName = host
	}
	tlsConn := tls.Client(conn, tlsClientConf)
	if err := tlsConn.Handshake(); err != nil {
		utils.IgnoreError(tlsConn.Close)
		return nil, err
	}
	return tlsConn, nil
}
