package compliance

import (
	"sync/atomic"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
)

// CommandHandler executes the input scrape commands, and reconciles scrapes with input ComplianceReturns,
// outputing the ScrapeUpdates we expect to be sent back to central.
type CommandHandler interface {
	Stopped() concurrency.ReadOnlyErrorSignal

	common.SensorComponent
}

// NewCommandHandler returns a new instance of a CommandHandler using the input image and Orchestrator.
func NewCommandHandler(complianceService Service) CommandHandler {
	reachable := &atomic.Bool{}
	reachable.Store(false)

	return &commandHandlerImpl{
		service: complianceService,

		commands: make(chan *central.ScrapeCommand),
		updates:  make(chan *message.ExpiringMessage),

		scrapeIDToState: make(map[string]*scrapeState),

		stopper:          concurrency.NewStopper(),
		centralReachable: reachable,
	}
}
