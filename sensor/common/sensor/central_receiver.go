package sensor

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/enforcers"
	complianceLogic "github.com/stackrox/rox/sensor/common/compliance"
	"github.com/stackrox/rox/sensor/common/networkpolicies"
)

// CentralReceiver handles receiving data from central.
type CentralReceiver interface {
	Start(stream central.SensorService_CommunicateClient, onStops ...func(error))

	Stop(err error)
	Stopped() concurrency.ReadOnlyErrorSignal
}

// NewCentralReceiver returns a new instance of a Receiver.
func NewCentralReceiver(scrapeCommandHandler complianceLogic.CommandHandler,
	enforcer enforcers.Enforcer,
	networkPoliciesCommandHandler networkpolicies.CommandHandler) CentralReceiver {
	return &centralReceiverImpl{
		scrapeCommandHandler:          scrapeCommandHandler,
		enforcer:                      enforcer,
		networkPoliciesCommandHandler: networkPoliciesCommandHandler,

		stopC:    concurrency.NewErrorSignal(),
		stoppedC: concurrency.NewErrorSignal(),
	}
}
