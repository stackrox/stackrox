package sensor

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

// CentralReceiver handles receiving data from central.
type CentralReceiver interface {
	Start(stream central.SensorService_CommunicateClient, onStops ...func())
	Stop()
	Stopped() concurrency.ReadOnlyErrorSignal
}

// NewCentralReceiver returns a new instance of a Receiver.
func NewCentralReceiver(finished *sync.WaitGroup, processor *ComponentProcessor) CentralReceiver {
	return &centralReceiverImpl{
		stopper:   concurrency.NewStopper(),
		processor: processor,
		finished:  finished,
	}
}
