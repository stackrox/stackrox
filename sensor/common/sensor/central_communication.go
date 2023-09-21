package sensor

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/detector"
	"google.golang.org/grpc"
)

// CentralCommunication interface allows you to start and stop the consumption/production loops.
type CentralCommunication interface {
	Start(centralConn grpc.ClientConnInterface, centralReachable *concurrency.Flag, handler config.Handler, detector detector.Detector)
	Stop(error)
	Stopped() concurrency.ReadOnlyErrorSignal
}

// NewCentralCommunication returns a new CentralCommunication.
func NewCentralCommunication(clientReconcile bool, reconnect bool, components ...common.SensorComponent) CentralCommunication {
	finished := sync.WaitGroup{}
	return &centralCommunicationImpl{
		allFinished: &finished,
		receiver:    NewCentralReceiver(&finished, components...),
		sender:      NewCentralSender(&finished, components...),
		components:  components,

		stopper:         concurrency.NewStopper(),
		isReconnect:     reconnect,
		clientReconcile: clientReconcile,
	}
}
