// Package vsockdialer dials KubeVirt VM VSOCK ports via the Kubernetes API.
//
// This replaces kubevirt.io/client-go's AsyncSubresourceHelper with a
// self-contained websocket dialer built on k8s.io/client-go/transport/websocket
// (the same building block client-go itself uses for `kubectl exec`/
// `port-forward` over WebSocket) plus gorilla/websocket for the connection
// type. We cannot import kubevirt.io/client-go directly yet — kubevirt#16951
// is no longer a blocker, but kubevirt#18382 and kubevirt#18408 still are.
// See https://github.com/stackrox/stackrox/pull/21587 for tracking; re-evaluate
// once both land in a tagged release. Do not switch to a personal fork for
// production use.
package vsockdialer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gorilla/websocket"
	"k8s.io/client-go/rest"
	wstransport "k8s.io/client-go/transport/websocket"
)

const (
	subresourceAPIGroup = "subresources.kubevirt.io"
	apiVersion          = "v1"
	wsSubprotocol       = "plain.kubevirt.io"
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
// deadline is propagated to the connection's read/write deadlines so the
// per-VM timeout budget covers dial + request + response.
//
// Deadline propagation alone does not abort an in-flight read when the
// parent context is cancelled without a nearer deadline (e.g. Sensor
// shutdown). GetReport closes the stream on ctx cancel for that case.
func (d *MultiDialer) Dial(ctx context.Context, namespace, name string, port uint32, useTLS bool) (io.ReadWriteCloser, error) {
	params := url.Values{}
	params.Set("port", strconv.FormatUint(uint64(port), 10))
	params.Set("tls", strconv.FormatBool(useTLS))

	wsURL, err := buildWSURL(d.config, "virtualmachineinstances", namespace, name, "vsock", params)
	if err != nil {
		return nil, fmt.Errorf("VSOCK dial %s/%s:%d: %w", namespace, name, port, err)
	}

	conn, err := dialSubresource(ctx, d.config, wsURL)
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

// dialSubresource negotiates a WebSocket connection to wsURL using
// k8s.io/client-go/transport/websocket. That package builds the
// RoundTripper straight from the REST config — TLS, proxying, and auth
// (bearer token, client certs, impersonation, exec/OIDC credential
// plugins) — the same way a real apiserver request would, so we don't
// have to reimplement auth-header extraction here.
//
// Trade-off: the negotiated connection's I/O buffers are fixed at ~33KiB by
// client-go's RoundTripper, smaller than a hand-rolled dialer could use.
// This only affects how many read/write syscalls a large message takes —
// messages are chunked across frames regardless of buffer size — so it
// doesn't affect correctness.
func dialSubresource(ctx context.Context, config *rest.Config, wsURL string) (*websocket.Conn, error) {
	rt, holder, err := wstransport.RoundTripperFor(config)
	if err != nil {
		return nil, fmt.Errorf("building websocket round tripper: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	conn, err := wstransport.Negotiate(rt, holder, req, wsSubprotocol)
	if err != nil {
		return nil, fmt.Errorf("websocket dial: %w", err)
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

	u = u.JoinPath("apis", subresourceAPIGroup, apiVersion, "namespaces", namespace, resource, name, subresource)
	if len(queryParams) > 0 {
		u.RawQuery = queryParams.Encode()
	}
	return u.String(), nil
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
	if _, ok := errors.AsType[*websocket.CloseError](err); ok {
		return true
	}
	return errors.Is(err, io.EOF)
}
