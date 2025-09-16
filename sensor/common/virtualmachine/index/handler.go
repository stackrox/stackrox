package index

import (
	"context"

	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
)

// Handler provides functionality to send virtual machine index reports to Central.
type Handler interface {
	common.SensorComponent

	Send(ctx context.Context, vm *v1.IndexReport) error
}

// NewHandler returns the virtual machine component for Sensor to use.
func NewHandler() Handler {
	return &handlerImpl{
		centralReady: concurrency.NewSignal(),
		lock:         &sync.RWMutex{},
		stopper:      concurrency.NewStopper(),
	}
}
