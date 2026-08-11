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
	return &MultiDialer{client: client}, nil
}

// Dial connects to the named VMI's VSOCK port in the given namespace.
// The context deadline is applied to the connection's read/write deadlines
// when present so the per-VM timeout budget covers dial + request + response.
//
// Deadline propagation alone does not abort an in-flight read when the
// parent context is cancelled without a nearer deadline (e.g. Sensor
// shutdown). GetReport closes the stream on ctx cancel for that case.
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
