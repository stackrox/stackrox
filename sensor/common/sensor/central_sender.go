package sensor

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
)

// CentralSender handles sending from sensor to central.
type CentralSender interface {
	Start(stream central.SensorService_CommunicateClient, onStops ...func(error))
	Stop(err error)
	Stopped() concurrency.ReadOnlyErrorSignal
}

// NewCentralSender returns a new instance of a CentralSender.
func NewCentralSender(senders ...common.SensorComponent) CentralSender {
	return &centralSenderImpl{
		stopC:    concurrency.NewErrorSignal(),
		stoppedC: concurrency.NewErrorSignal(),

		senders: senders,
	}
}
