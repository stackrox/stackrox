package compliance

import (
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/sensor/common"
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

		stopC:    concurrency.NewErrorSignal(),
		stoppedC: concurrency.NewErrorSignal(),
	}
}
