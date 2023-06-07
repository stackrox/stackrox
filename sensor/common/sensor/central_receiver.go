package sensor

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
)

// CentralReceiver handles receiving data from central.
type CentralReceiver interface {
	Start(stream central.SensorService_CommunicateClient, onStops ...func(error))
	Stop(err error)
	Stopped() concurrency.ReadOnlyErrorSignal
}

// NewCentralReceiver returns a new instance of a Receiver.
func NewCentralReceiver(finished *sync.WaitGroup, receivers ...common.SensorComponent) CentralReceiver {
	return &centralReceiverImpl{
		stopper:   concurrency.NewStopper(),
		receivers: receivers,
		finished:  finished,
	}
}
