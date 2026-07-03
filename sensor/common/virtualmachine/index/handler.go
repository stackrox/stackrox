package index

import (
	"context"
	"time"

	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/virtualmachine"
)

// Handler provides functionality to send virtual machine index reports to Central.
// It embeds ComplianceComponent (which itself embeds SensorComponent) so that
// compliance channel wiring is part of the compile-time contract.
type Handler interface {
	common.ComplianceComponent

	// Send enqueues a report for delivery to Central. Delivery is Fair FIFO
	// across VMs: reports are handed to Central in the order their VM first
	// queued one, regardless of trigger type. generatedAt is roxagent's
	// report-generation timestamp (ResponseMeta.report_generated_at) for a
	// reactive (event-triggered) report, or the zero value for a routine
	// scheduled one; it is used only to measure Sensor-side reactive-update
	// delivery latency and is never forwarded to Central. If this VM
	// already has a not-yet-delivered report queued, the new one replaces
	// it in place (coalescing) rather than queuing alongside it or moving
	// its position in the queue — only the latest report for a VM is ever
	// meaningful.
	Send(ctx context.Context, vm *v1.IndexReport, generatedAt time.Time) error
}

// VirtualMachineStore interface to the VirtualMachine store
//
//go:generate mockgen-wrapper
type VirtualMachineStore interface {
	Get(id virtualmachine.VMID) *virtualmachine.Info
	GetFromCID(cid uint32) *virtualmachine.Info
}

// NewHandler returns the virtual machine component for Sensor to use.
func NewHandler(store VirtualMachineStore) Handler {
	return &handlerImpl{
		centralReady: concurrency.NewSignal(),
		lock:         &sync.RWMutex{},
		stopper:      concurrency.NewStopper(),
		store:        store,
	}
}
