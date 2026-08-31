// Package vsockdialer dials KubeVirt VM VSOCK ports via the Kubernetes API.
//
// It uses kubevirt client-go's AsyncSubresourceHelper so Sensor can keep a
// small Dial(namespace, name, port) surface for the pull-mode scraper.
package vsockdialer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/stackrox/rox/pkg/k8sutil"
	"k8s.io/client-go/rest"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"
)

// MultiDialer dials VMs across namespaces via the KubeVirt subresource API.
type MultiDialer struct {
	config *rest.Config
}

// NewMultiDialer creates a dialer from in-cluster Kubernetes config.
func NewMultiDialer() (*MultiDialer, error) {
	config, err := k8sutil.GetK8sInClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("loading in-cluster config: %w", err)
	}
	return NewMultiDialerFromConfig(config), nil
}

// NewMultiDialerFromConfig builds a dialer from an existing REST config so
// tests and local runs can skip in-cluster service-account lookup.
func NewMultiDialerFromConfig(config *rest.Config) *MultiDialer {
	return &MultiDialer{config: config}
}

// Dial connects to the named VMI's VSOCK port in the given namespace.
// ctx's deadline is applied to the established connection only: the
// WebSocket handshake does not take ctx, so it cannot be cancelled.
func (d *MultiDialer) Dial(ctx context.Context, namespace, name string, port uint32, useTLS bool) (io.ReadWriteCloser, error) {
	params := url.Values{}
	params.Set("port", strconv.FormatUint(uint64(port), 10))
	params.Set("tls", strconv.FormatBool(useTLS))
	stream, err := kvcorev1.AsyncSubresourceHelper(
		d.config, "virtualmachineinstances", namespace, name, "vsock", params,
	)
	if err != nil {
		return nil, fmt.Errorf("VSOCK dial %s/%s:%d: %w", namespace, name, port, err)
	}
	conn := stream.AsConn()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	return &closeCodeConn{Conn: conn}, nil
}

// closeCodeConn remaps kubevirt AsConn() close errors into closedError so
// callers can recover the websocket close code via errors.As against their
// own CloseCode() interface, without importing this package.
type closeCodeConn struct {
	net.Conn
}

func (c *closeCodeConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if closeErr := transportClosedErr(err); closeErr != nil {
		return n, closeErr
	}
	return n, err //nolint:wrapcheck // implements io.Reader
}

func (c *closeCodeConn) Write(p []byte) (int, error) {
	n, err := c.Conn.Write(p)
	if closeErr := transportClosedErr(err); closeErr != nil {
		return n, closeErr
	}
	return n, err //nolint:wrapcheck // implements io.Writer
}

// transportClosedErr returns a *closedError if err represents a closed
// connection, or nil if it's some other read failure.
func transportClosedErr(err error) *closedError {
	if err == nil {
		return nil
	}
	if ce, ok := errors.AsType[*websocket.CloseError](err); ok {
		return &closedError{code: ce.Code, reason: ce.Text}
	}
	if errors.Is(err, io.EOF) {
		return &closedError{}
	}
	return nil
}

// closedError indicates the websocket connection closed before a complete
// VMServiceResponse could be read. Kept unexported and dependency-free so
// callers recover it structurally (via errors.As against their own
// CloseCode() interface) rather than importing this package.
type closedError struct {
	code   int
	reason string
}

func (e *closedError) Error() string {
	if e.code == 0 {
		return "websocket connection closed"
	}
	return fmt.Sprintf("websocket connection closed (code %d): %s", e.code, e.reason)
}

// Is reports whether this close is an io.EOF-equivalent (no code, or a
// normal 1000 close). Abnormal codes such as 1006 are not EOF.
func (e *closedError) Is(target error) bool {
	if target != io.EOF {
		return false
	}
	return e.code == 0 || e.code == websocket.CloseNormalClosure
}

// CloseCode returns the close code and reason; code is 0 if unstructured.
func (e *closedError) CloseCode() (code int, reason string) {
	return e.code, e.reason
}
