package virtualmachine

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
)

// Component provides functionality to send virtual machines to Central.
type Component interface {
	common.SensorComponent

	Send(ctx context.Context, vm *storage.VirtualMachine) error
}

// NewComponent returns the virtual machine component for Sensor to use.
func NewComponent() Component {
	return &componentImpl{
		centralReady:    concurrency.NewSignal(),
		lock:            &sync.Mutex{},
		stopper:         concurrency.NewStopper(),
		virtualMachines: make(chan *storage.VirtualMachine),
	}
}
