package sensor

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/enforcers"
	"github.com/stackrox/rox/pkg/listeners"
	"github.com/stackrox/rox/sensor/common/clusterstatus"
	complianceLogic "github.com/stackrox/rox/sensor/common/compliance"
	"github.com/stackrox/rox/sensor/common/config"
	networkConnManager "github.com/stackrox/rox/sensor/common/networkflow/manager"
	"github.com/stackrox/rox/sensor/common/networkpolicies"
	"github.com/stackrox/rox/sensor/common/signal"
	"google.golang.org/grpc"
)

// CentralCommunication interface allows you to start and stop the consumption/production loops.
type CentralCommunication interface {
	Start(centralConn *grpc.ClientConn, centralReachable *concurrency.Flag, handler config.Handler)

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
	networkPoliciesCommandHandler networkpolicies.CommandHandler,
	clusterStatusUpdater clusterstatus.Updater,
	configCommandHandler config.Handler) CentralCommunication {
	return &centralCommunicationImpl{
		receiver: NewCentralReceiver(scrapeCommandHandler, enforcer, networkPoliciesCommandHandler, configCommandHandler),
		sender:   NewCentralSender(listener, signalService, networkConnManager, scrapeCommandHandler, networkPoliciesCommandHandler, clusterStatusUpdater, configCommandHandler),

		stopC:    concurrency.NewErrorSignal(),
		stoppedC: concurrency.NewErrorSignal(),
	}
}
