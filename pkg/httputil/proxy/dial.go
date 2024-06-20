package proxy

import (
	"context"
	"net"
	"net/url"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/utils"
	"golang.org/x/net/proxy"
)

const (
	defaultSOCKS5Port = "1080"
)

// DialWithProxy performs a context-aware TCP dial, possibly using the given proxy URL if non-nil. It supports
// http(s) (via CONNECT) and socks5 proxy URLs.
func DialWithProxy(ctx context.Context, proxyURL *url.URL, address string) (net.Conn, error) {
	if proxyURL == nil {
		conn, err := defaultDialer.DialContext(ctx, "tcp", address)
		return conn, errox.ConcealSensitive(err)
	}

	switch proxyURL.Scheme {
	case "http", "https":
		return dialWithConnectProxy(ctx, proxyURL, address)
	case "socks5", "socks5h":
		return dialWithSocks5Proxy(ctx, proxyURL, address)
	default:
		return nil, errors.Errorf("invalid scheme in proxy URL")
	}
}

func dialWithSocks5Proxy(ctx context.Context, proxyURL *url.URL, address string) (net.Conn, error) {
	var auth *proxy.Auth
	if proxyURL.User != nil {
		auth = &proxy.Auth{
			User: proxyURL.User.Username(),
		}
		if p, ok := proxyURL.User.Password(); ok {
			auth.Password = p
		}
	}

	host, zone, port, err := netutil.ParseEndpoint(proxyURL.Host)
	if err != nil {
		// parsing err has no sensitive information, but the added context has:
		return nil, errox.NewSensitive(
			errox.WithSensitivef("invalid endpoint in proxy URL %q", proxyURL),
			errox.WithPublicMessage("invalid endpoint in proxy URL"),
			errox.WithPublicError(err),
		)
	}
	if port == "" {
		port = defaultSOCKS5Port
	}
	endpoint := netutil.FormatEndpoint(host, zone, port)
	socksDialer, err := proxy.SOCKS5("tcp", endpoint, auth, defaultDialer)
	if err != nil {
		return nil, errors.Wrapf(errox.ConcealSensitive(err), "failed to create SOCKS5 proxy dialer")
	}
	socksCtxDialer, _ := socksDialer.(proxy.ContextDialer)
	if socksCtxDialer == nil {
		return nil, utils.ShouldErr(errors.New("expected SOCKS5 dialer to implement DialContext"))
	}
	conn, err := socksCtxDialer.DialContext(ctx, "tcp", address)
	return conn, errox.ConcealSensitive(err)
}
