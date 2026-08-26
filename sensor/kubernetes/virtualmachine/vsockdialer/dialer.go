// Package vsockdialer dials KubeVirt VM VSOCK ports via the Kubernetes API.
//
// It uses kubevirt client-go's AsyncSubresourceHelper so Sensor can keep a
// small Dial(namespace, name, port) surface for the pull-mode scraper.
package vsockdialer

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"

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
	return conn, nil
}
