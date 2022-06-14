package proxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/netutil"
)

// readHeaderBytes reads from the given reader byte by byte(!) until the end of header sequence (\r\n\r\n) is found.
// While inefficient, this is done to avoid reading anything off the reader that is not part of the header.
// Note that we cannot make any assumptions about the server, i.e., there is no guarantee it will *not* start talking
// right away, which might be picked up by a bufio.Reader.
func readHeaderBytes(r io.Reader) ([]byte, error) {
	var resp bytes.Buffer
	var buf [1]byte
	for {
		_, err := io.ReadFull(r, buf[:])
		if err != nil {
			return nil, err
		}
		if err := resp.WriteByte(buf[0]); err != nil {
			return nil, err
		}
		if buf[0] == '\n' && bytes.HasSuffix(resp.Bytes(), []byte("\r\n\r\n")) {
			return resp.Bytes(), nil
		}
	}
}

var (
	defaultPortsByScheme = map[string]string{
		"http":  "80",
		"https": "443",
	}
)

func dialWithConnectProxy(ctx context.Context, proxyURL *url.URL, address string) (net.Conn, error) {
	proxyAddress := proxyURL.Host
	host, zone, port, err := netutil.ParseEndpoint(proxyAddress)
	if err != nil {
		return nil, errors.Wrapf(err, "unparseable proxy address %q", proxyAddress)
	}
	if port == "" {
		defPort, ok := defaultPortsByScheme[proxyURL.Scheme]
		if !ok {
			return nil, errors.Errorf("invalid scheme %q in proxy URL", proxyURL.Scheme)
		}
		port = defPort
	}
	proxyAddress = netutil.FormatEndpoint(host, zone, port)

	rawConn, err := defaultDialer.DialContext(ctx, "tcp", proxyAddress)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to dial proxy %q", proxyAddress)
	}
	closeOnErrConn := rawConn
	defer func() {
		if closeOnErrConn != nil {
			_ = closeOnErrConn.Close()
		}
	}()

	if proxyURL.Scheme == "https" {
		rawConn = tls.Client(rawConn, &tls.Config{})
	}

	// Note: the URL in the next line only matters for making sure we sent a correct `Host:` header.
	// We override the actual request URL via the `Opaque` field afterwards.
	connectRequest, err := http.NewRequestWithContext(ctx, http.MethodConnect, fmt.Sprintf("https://%s", proxyAddress), nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not create CONNECT request")
	}
	connectRequest.URL.Opaque = address
	if connectRequest.Header == nil {
		connectRequest.Header = make(http.Header)
	}
	connectRequest.Header.Set("Proxy-Connection", "Keep-Alive")
	authStr := proxyURL.User.String()
	if authStr != "" {
		connectRequest.Header.Set("Proxy-Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(authStr))))
	}

	if err := connectRequest.WriteProxy(rawConn); err != nil {
		return nil, errors.Wrap(err, "error sending connect request to server")
	}

	headerBytes, err := readHeaderBytes(rawConn)
	if err != nil {
		return nil, errors.Wrap(err, "reading proxy response headers")
	}
	resp, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(headerBytes)), connectRequest)
	if err != nil {
		return nil, errors.Wrap(err, "parsing proxy response")
	}
	_ = resp.Body.Close()
	if !httputil.Is2xxStatusCode(resp.StatusCode) {
		return nil, errors.Errorf("proxy CONNECT returned status code %d", resp.StatusCode)
	}

	closeOnErrConn = nil
	return rawConn, nil
}
