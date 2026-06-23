package vsockclient

import (
	"fmt"
	"net/url"
	"strconv"

	"k8s.io/client-go/rest"
	kubevirtv1 "kubevirt.io/api/core/v1"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"
)

// DefaultVSOCKPort is the port roxagent listens on inside the VM.
const DefaultVSOCKPort uint32 = 818

// VMIVSOCKer is the subset of the KubeVirt client we need. Satisfied by
// kubecli.KubevirtClient (via VirtualMachineInstance(ns)).
type VMIVSOCKer interface {
	VSOCK(name string, options *kubevirtv1.VSOCKOptions) (kvcorev1.StreamInterface, error)
}

// Dialer connects to a VM's VSOCK port via the KubeVirt API.
type Dialer struct {
	vmiClient VMIVSOCKer
}

// NewDialer creates a Dialer from a VMIVSOCKer (typically the result of
// kubecli.KubevirtClient.VirtualMachineInstance(namespace)).
func NewDialer(vmiClient VMIVSOCKer) *Dialer {
	return &Dialer{vmiClient: vmiClient}
}

// Dial connects to the given VMI's VSOCK port and returns a stream.
// The request path: Sensor -> virt-api -> virt-handler (on VM's node) -> vsock into guest.
// The caller must close the returned stream.
// useTLS=false for the POC; TLS validation is a follow-up.
func (d *Dialer) Dial(vmiName string, port uint32, useTLS bool) (StreamReader, error) {
	opts := &kubevirtv1.VSOCKOptions{
		TargetPort: port,
		UseTLS:     &useTLS,
	}

	stream, err := d.vmiClient.VSOCK(vmiName, opts)
	if err != nil {
		return nil, fmt.Errorf("VSOCK dial to %s:%d: %w", vmiName, port, err)
	}

	// ponytail: AsConn() wraps the existing websocket as a net.Conn —
	// cheap, no error path. net.Conn satisfies StreamReader (io.Reader + Close).
	return stream.AsConn(), nil
}

// MultiDialer dials VMs across namespaces using the KubeVirt API directly.
// This bypasses kubecli (which has transitive dep issues with storagemigration
// mocks) and calls AsyncSubresourceHelper on the typed client layer.
type MultiDialer struct {
	config *rest.Config
}

// NewMultiDialer creates a dialer from in-cluster (or kubeconfig) REST config.
func NewMultiDialer(config *rest.Config) *MultiDialer {
	return &MultiDialer{config: config}
}

// Dial connects to the named VMI's VSOCK port in the given namespace.
func (d *MultiDialer) Dial(namespace, name string, port uint32, useTLS bool) (StreamReader, error) {
	params := url.Values{}
	params.Set("port", strconv.FormatUint(uint64(port), 10))
	params.Set("tls", strconv.FormatBool(useTLS))

	stream, err := kvcorev1.AsyncSubresourceHelper(
		d.config, "virtualmachineinstances", namespace, name, "vsock", params,
	)
	if err != nil {
		return nil, fmt.Errorf("VSOCK dial %s/%s:%d: %w", namespace, name, port, err)
	}

	return stream.AsConn(), nil
}
