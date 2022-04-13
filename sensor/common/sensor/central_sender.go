package sensor

import (
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/sensor/common"
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
