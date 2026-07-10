// Package vsockdialer dials KubeVirt VM VSOCK ports via KubeVirt's own
// client (kubevirt.io/client-go/kubecli), which implements the
// virtualmachineinstances/vsock subresource WebSocket handshake for us.
//
// This used to be a self-contained websocket dialer using only
// k8s.io/client-go/rest and gorilla/websocket, because kubevirt.io/client-go
// couldn't be imported: its log package unconditionally registered a -v flag
// in init(), which panicked when glog (already in the sensor dep tree) also
// registered -v. That upstream bug is fixed (private FlagSet + VerbosityFlag()),
// see https://github.com/kubevirt/kubevirt/pull/18294, so we now depend on the
// real client instead of reimplementing its dialer.
//
// TODO(ROX-35192): the fix hasn't synced to github.com/kubevirt/client-go yet,
// so go.mod currently points kubevirt.io/client-go at a local/forked copy carrying
// it cherry-picked. Swap back to the real module once the mirror catches up.
package vsockdialer

import (
	"context"
	"fmt"
	"io"

	"k8s.io/client-go/rest"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

// MultiDialer dials VMs across namespaces via the KubeVirt subresource API.
type MultiDialer struct {
	client      kubecli.KubevirtClient
	wsReadLimit int64
}

// NewMultiDialer creates a dialer from in-cluster (or kubeconfig) REST config.
// wsReadLimit sets the maximum WebSocket message size in bytes.
func NewMultiDialer(config *rest.Config, wsReadLimit int64) (*MultiDialer, error) {
	client, err := kubecli.GetKubevirtClientFromRESTConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating KubeVirt client: %w", err)
	}
	return &MultiDialer{client: client, wsReadLimit: wsReadLimit}, nil
}

// readLimiter is implemented by the *websocket.Conn embedded in the net.Conn
// that kubecli's StreamInterface.AsConn() returns, but isn't part of the
// net.Conn interface itself, hence the type assertion in Dial.
type readLimiter interface {
	SetReadLimit(limit int64)
}

// Dial connects to the named VMI's VSOCK port in the given namespace.
// The context does not control the dial itself (kubecli's VSOCK call takes
// no context), but if it carries a deadline, that deadline is propagated to
// the connection's read/write deadlines so the request/response exchange is
// still bounded.
func (d *MultiDialer) Dial(ctx context.Context, namespace, name string, port uint32, useTLS bool) (io.ReadWriteCloser, error) {
	stream, err := d.client.VirtualMachineInstance(namespace).VSOCK(name, &v1.VSOCKOptions{
		TargetPort: port,
		UseTLS:     &useTLS,
	})
	if err != nil {
		return nil, fmt.Errorf("VSOCK dial %s/%s:%d: %w", namespace, name, port, err)
	}

	conn := stream.AsConn()
	if limiter, ok := conn.(readLimiter); ok {
		limiter.SetReadLimit(d.wsReadLimit)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetReadDeadline(deadline)
		_ = conn.SetWriteDeadline(deadline)
	}
	return conn, nil
}
