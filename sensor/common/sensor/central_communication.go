package sensor

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/enforcers"
	"github.com/stackrox/rox/pkg/listeners"
	"github.com/stackrox/rox/sensor/common/compliance"
	complianceLogic "github.com/stackrox/rox/sensor/common/compliance"
	networkConnManager "github.com/stackrox/rox/sensor/common/networkflow/manager"
	"github.com/stackrox/rox/sensor/common/signal"
	"google.golang.org/grpc"
)

// CentralCommunication interface allows you to start and stop the consumption/production loops.
type CentralCommunication interface {
	Start(centralConn *grpc.ClientConn)

	Stop(error)
	Stopped() concurrency.ReadOnlyErrorSignal
}

// NewCentralCommunication returns a new CentralCommunication.
func NewCentralCommunication(
	scrapeCommandHandler complianceLogic.CommandHandler,
	enforcer enforcers.Enforcer,
	listener listeners.Listener,
	signalService signal.Service,
	networkConnManager networkConnManager.Manager,
	scrapeReceiver compliance.Service) CentralCommunication {
	return &centralCommunicationImpl{
		receiver: NewCentralReceiver(scrapeCommandHandler, enforcer),
		sender:   NewCentralSender(listener, signalService, networkConnManager, scrapeReceiver),

		stopC:    concurrency.NewErrorSignal(),
		stoppedC: concurrency.NewErrorSignal(),
	}
}
