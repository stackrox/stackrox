// Package vsockdialer dials KubeVirt VM VSOCK ports via the Kubernetes API.
//
// This replaces kubevirt.io/client-go's AsyncSubresourceHelper with a
// self-contained websocket dialer using only k8s.io/client-go/rest and
// gorilla/websocket. We cannot import kubevirt.io/client-go because its
// log package unconditionally registers a -v flag in init(), which panics
// when glog (already in the sensor dep tree) also registers -v.
// Upstream code: https://github.com/kubevirt/kubevirt/blob/main/staging/src/kubevirt.io/client-go/log/log.go#L88
// Upstream bug:  https://github.com/kubevirt/kubevirt/issues/16951
package vsockdialer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"

	"github.com/gorilla/websocket"
	"k8s.io/client-go/rest"
)

const (
	subresourceAPIGroup = "subresources.kubevirt.io"
	apiVersion          = "v1"
	wsSubprotocol       = "plain.kubevirt.io"
	wsBufferSize        = 1 << 20 // 1 MiB I/O buffer; not a message size limit (larger messages are chunked internally)
)

// MultiDialer dials VMs across namespaces via the KubeVirt subresource API.
type MultiDialer struct {
	config      *rest.Config
	wsReadLimit int64
}

// NewMultiDialer creates a dialer from in-cluster (or kubeconfig) REST config.
// wsReadLimit sets the maximum WebSocket message size in bytes.
func NewMultiDialer(config *rest.Config, wsReadLimit int64) *MultiDialer {
	return &MultiDialer{config: config, wsReadLimit: wsReadLimit}
}

// Dial connects to the named VMI's VSOCK port in the given namespace.
// The context controls dial timeout and, if it carries a deadline, that
// deadline is propagated to the connection's read/write deadlines so
// the entire operation (dial + request + response) is bounded.
//
// An alternative would be threading context.Context through GetReport
// with goroutine+select cancellation — we chose deadline propagation
// for simplicity since the scraper's per-VM context already expresses
// the right timeout budget for the whole exchange.
func (d *MultiDialer) Dial(ctx context.Context, namespace, name string, port uint32, useTLS bool) (io.ReadWriteCloser, error) {
	params := url.Values{}
	params.Set("port", strconv.FormatUint(uint64(port), 10))
	params.Set("tls", strconv.FormatBool(useTLS))

	conn, err := dialSubresource(ctx, d.config, "virtualmachineinstances", namespace, name, "vsock", params)
	if err != nil {
		return nil, fmt.Errorf("VSOCK dial %s/%s:%d: %w", namespace, name, port, err)
	}
	conn.SetReadLimit(d.wsReadLimit)
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetReadDeadline(deadline)
		_ = conn.SetWriteDeadline(deadline)
	}
	return &wsStream{conn: conn}, nil
}

func dialSubresource(ctx context.Context, config *rest.Config, resource, namespace, name, subresource string, queryParams url.Values) (*websocket.Conn, error) {
	wsURL, err := buildWSURL(config, resource, namespace, name, subresource, queryParams)
	if err != nil {
		return nil, err
	}

	headers, err := authHeaders(config)
	if err != nil {
		return nil, err
	}

	tlsConfig, err := rest.TLSConfigFor(config)
	if err != nil {
		return nil, fmt.Errorf("TLS config: %w", err)
	}

	proxy := http.ProxyFromEnvironment
	if config.Proxy != nil {
		proxy = config.Proxy
	}

	dialer := &websocket.Dialer{
		Proxy:           proxy,
		TLSClientConfig: tlsConfig,
		WriteBufferSize: wsBufferSize,
		ReadBufferSize:  wsBufferSize,
		Subprotocols:    []string{wsSubprotocol},
		NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, addr)
		},
	}

	conn, resp, err := dialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		code := 0
		if resp != nil {
			code = resp.StatusCode
		}
		return nil, fmt.Errorf("websocket dial (status %d): %w", code, err)
	}
	return conn, nil
}

func buildWSURL(config *rest.Config, resource, namespace, name, subresource string, queryParams url.Values) (string, error) {
	u, err := url.Parse(config.Host)
	if err != nil {
		return "", fmt.Errorf("parsing host: %w", err)
	}

	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	default:
		return "", fmt.Errorf("unsupported scheme %q", u.Scheme)
	}

	u.Path = path.Join(u.Path,
		fmt.Sprintf("/apis/%s/%s/namespaces/%s/%s/%s/%s",
			subresourceAPIGroup, apiVersion, namespace, resource, name, subresource))
	if len(queryParams) > 0 {
		u.RawQuery = queryParams.Encode()
	}
	return u.String(), nil
}

// authHeaders extracts auth headers (bearer token, client certs, impersonation)
// that rest.HTTPWrappersForConfig injects into requests.
func authHeaders(config *rest.Config) (http.Header, error) {
	capture := &headerCapture{}
	rt, err := rest.HTTPWrappersForConfig(config, capture)
	if err != nil {
		return nil, fmt.Errorf("wrapping transport: %w", err)
	}
	probe, err := http.NewRequest(http.MethodGet, config.Host, nil)
	if err != nil {
		return nil, fmt.Errorf("building probe request: %w", err)
	}
	_, _ = rt.RoundTrip(probe)
	return capture.headers, nil
}

type headerCapture struct{ headers http.Header }

func (h *headerCapture) RoundTrip(req *http.Request) (*http.Response, error) {
	h.headers = req.Header.Clone()
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
}

// wsStream adapts a websocket.Conn into an io.ReadWriteCloser that reads
// across websocket binary message boundaries and writes binary messages.
type wsStream struct {
	conn   *websocket.Conn
	reader io.Reader
}

func (s *wsStream) Read(p []byte) (int, error) {
	for {
		if s.reader == nil {
			msgType, rd, err := s.conn.NextReader()
			if err != nil {
				if isWSClose(err) {
					return 0, io.EOF
				}
				return 0, err //nolint:wrapcheck // implements io.Reader
			}
			if msgType == websocket.CloseMessage {
				return 0, io.EOF
			}
			s.reader = rd
		}

		n, err := s.reader.Read(p)
		if err == io.EOF {
			s.reader = nil
			if n > 0 {
				return n, nil
			}
			continue
		}
		return n, err //nolint:wrapcheck // implements io.Reader
	}
}

func (s *wsStream) Write(p []byte) (int, error) {
	if err := s.conn.WriteMessage(websocket.BinaryMessage, p); err != nil {
		return 0, fmt.Errorf("writing websocket message: %w", err)
	}
	return len(p), nil
}

func (s *wsStream) Close() error {
	if err := s.conn.Close(); err != nil {
		return fmt.Errorf("closing websocket: %w", err)
	}
	return nil
}

func isWSClose(err error) bool {
	var ce *websocket.CloseError
	if errors.As(err, &ce) {
		return true
	}
	return errors.Is(err, io.EOF)
}
