package proxy

import (
	"context"
	"net"
	"net/url"

	"github.com/pkg/errors"
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
		return defaultDialer.DialContext(ctx, "tcp", address)
	}

	switch proxyURL.Scheme {
	case "http", "https":
		return dialWithConnectProxy(ctx, proxyURL, address)
	case "socks5", "socks5h":
		return dialWithSocks5Proxy(ctx, proxyURL, address)
	default:
		return nil, errors.Errorf("invalid scheme %q in proxy URL %v", proxyURL.Scheme, proxyURL)
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
		return nil, errors.Wrapf(err, "invalid endpoint in proxy URL %v", proxyURL)
	}
	if port == "" {
		port = defaultSOCKS5Port
	}
	endpoint := netutil.FormatEndpoint(host, zone, port)
	socksDialer, err := proxy.SOCKS5("tcp", endpoint, auth, defaultDialer)
	if err != nil {
		return nil, err
	}
	socksCtxDialer, _ := socksDialer.(proxy.ContextDialer)
	if socksCtxDialer == nil {
		return nil, utils.ShouldErr(errors.New("expected SOCKS5 dialer to implement DialContext"))
	}
	return socksCtxDialer.DialContext(ctx, "tcp", address)
}
