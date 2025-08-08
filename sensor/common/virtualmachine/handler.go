package virtualmachine

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
)

// Handler provides functionality to send virtual machines to Central.
type Handler interface {
	common.SensorComponent

	Send(ctx context.Context, vm *sensor.VirtualMachine) error
}

// NewHandler returns the virtual machine component for Sensor to use.
func NewHandler() Handler {
	return &handlerImpl{
		centralReady: concurrency.NewSignal(),
		lock:         &sync.RWMutex{},
		stopper:      concurrency.NewStopper(),
	}
}
