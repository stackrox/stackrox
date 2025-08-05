package virtualmachine

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
)

const virtualMachineBufferedChannelSize = 100

// Component provides functionality to send virtual machines to Central.
type Component interface {
	common.SensorComponent

	Send(ctx context.Context, vm *sensor.VirtualMachine) error
}

// NewComponent returns the virtual machine component for Sensor to use.
func NewComponent() Component {
	return &componentImpl{
		centralReady:    concurrency.NewSignal(),
		lock:            &sync.Mutex{},
		stopper:         concurrency.NewStopper(),
		virtualMachines: make(chan *sensor.VirtualMachine, virtualMachineBufferedChannelSize),
	}
}
