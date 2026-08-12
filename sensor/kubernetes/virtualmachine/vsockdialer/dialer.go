// Package vsockdialer dials KubeVirt VM VSOCK ports via the Kubernetes API.
//
// It wraps kubevirt.io/client-go's kubecli VSOCK helper so Sensor can keep a
// small Dial(namespace, name, port) surface for the pull-mode scraper.
package vsockdialer

import (
	"context"
	"fmt"
	"io"

	"github.com/stackrox/rox/pkg/k8sutil"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

// MultiDialer dials VMs across namespaces via the KubeVirt subresource API.
type MultiDialer struct {
	client kubecli.KubevirtClient
}

// NewMultiDialer creates a dialer from in-cluster Kubernetes config.
func NewMultiDialer() (*MultiDialer, error) {
	config, err := k8sutil.GetK8sInClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("loading in-cluster config: %w", err)
	}
	client, err := kubecli.GetKubevirtClientFromRESTConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating kubevirt client: %w", err)
	}
	return NewMultiDialerFromClient(client), nil
}

// NewMultiDialerFromClient creates a dialer from an existing KubeVirt client.
func NewMultiDialerFromClient(client kubecli.KubevirtClient) *MultiDialer {
	return &MultiDialer{client: client}
}

// Dial connects to the named VMI's VSOCK port in the given namespace.
// When ctx carries a deadline, it is applied to the established connection's
// read/write deadlines only. kubecli's VSOCK path does not take ctx into the
// WebSocket handshake, so cancellation/deadline do not abort dial or upgrade.
//
// GetReport closes the stream on ctx cancel so an in-flight read can still
// stop when the parent context ends without a nearer deadline (e.g. Sensor
// shutdown).
func (d *MultiDialer) Dial(ctx context.Context, namespace, name string, port uint32, useTLS bool) (io.ReadWriteCloser, error) {
	stream, err := d.client.VirtualMachineInstance(namespace).VSOCK(name, &v1.VSOCKOptions{
		TargetPort: port,
		UseTLS:     &useTLS,
	})
	if err != nil {
		return nil, fmt.Errorf("VSOCK dial %s/%s:%d: %w", namespace, name, port, err)
	}
	conn := stream.AsConn()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetReadDeadline(deadline)
		_ = conn.SetWriteDeadline(deadline)
	}
	return conn, nil
}
