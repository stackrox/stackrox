package sensor

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/deduper"
)

// CentralSender handles sending from sensor to central.
type CentralSender interface {
	Start(stream central.SensorService_CommunicateClient, initialDeduperState map[deduper.Key]uint64, onStops ...func(error))
	Stop(err error)
	Stopped() concurrency.ReadOnlyErrorSignal
	OnSync(func())
}

// NewCentralSender returns a new instance of a CentralSender.
func NewCentralSender(finished *sync.WaitGroup, senders ...common.SensorComponent) CentralSender {
	return &centralSenderImpl{
		stopper:  concurrency.NewStopper(),
		senders:  senders,
		finished: finished,
	}
}
