package compliance

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/orchestrators"
)

// CommandHandler executes the input scrape commands, and reconciles scrapes with input ComplianceReturns,
// outputing the ScrapeUpdates we expect to be sent back to central.
type CommandHandler interface {
	Start(<-chan *compliance.ComplianceReturn)
	Stop(err error)
	Stopped() concurrency.ReadOnlyErrorSignal

	SendCommand(*central.ScrapeCommand) bool
	Output() <-chan *central.ScrapeUpdate
}

// NewCommandHandler returns a new instance of a CommandHandler using the input image and Orchestrator.
func NewCommandHandler(image string, orchestrator orchestrators.Orchestrator) CommandHandler {
	return &commandHandlerImpl{
		image:        image,
		orchestrator: orchestrator,

		commands: make(chan *central.ScrapeCommand),
		updates:  make(chan *central.ScrapeUpdate),

		scrapeIDToState: make(map[string]*scrapeState),

		stopC:    concurrency.NewErrorSignal(),
		stoppedC: concurrency.NewErrorSignal(),
	}
}
