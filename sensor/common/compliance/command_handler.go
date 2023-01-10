package compliance

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
)

// CommandHandler executes the input scrape commands, and reconciles scrapes with input ComplianceReturns,
// outputing the ScrapeUpdates we expect to be sent back to central.
type CommandHandler interface {
	Stopped() concurrency.ReadOnlyErrorSignal

	common.SensorComponent
}

// NewCommandHandler returns a new instance of a CommandHandler using the input image and Orchestrator.
func NewCommandHandler(complianceService Service) CommandHandler {
	return &commandHandlerImpl{
		service: complianceService,

		commands: make(chan *central.ScrapeCommand),
		updates:  make(chan *central.MsgFromSensor),

		scrapeIDToState: make(map[string]*scrapeState),

		stopper: concurrency.NewStopper(),
	}
}
