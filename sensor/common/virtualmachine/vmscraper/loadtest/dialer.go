package loadtest

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"
)

// FarmDialer implements vmscraper.VMDialer by connecting the caller directly
// to an in-process FarmVM's Handler over a net.Pipe.
//
// intentional simplification: this is the "pipe" transport mode only (zero
// network/OS overhead, pure Sensor+protocol-code stress test). The "uds" and
// "tcp" modes from the design spec are a later iteration, added once this
// mode has validated the overall approach end-to-end.
type FarmDialer struct {
	farm    *Farm
	latency time.Duration
}

// NewFarmDialer creates a dialer backed by farm. latency is an artificial
// per-dial delay simulating KubeVirt/VSOCK round-trip overhead (0 = none).
func NewFarmDialer(farm *Farm, latency time.Duration) *FarmDialer {
	return &FarmDialer{farm: farm, latency: latency}
}

// Dial implements vmscraper.VMDialer. Port and TLS are accepted for interface
// compatibility but unused: the farm has no network/TLS layer to configure.
func (d *FarmDialer) Dial(ctx context.Context, namespace, name string, _ uint32, _ bool) (io.ReadWriteCloser, error) {
	vm := d.farm.Get(namespace, name)
	if vm == nil {
		return nil, fmt.Errorf("loadtest: no synthetic VM %s/%s", namespace, name)
	}

	if d.latency > 0 {
		select {
		case <-time.After(d.latency):
		case <-ctx.Done():
			return nil, fmt.Errorf("loadtest: dial to %s/%s cancelled: %w", namespace, name, ctx.Err())
		}
	}

	client, server := net.Pipe()
	go vm.Handler.HandleConn(server)
	vm.markScraped()
	return client, nil
}
