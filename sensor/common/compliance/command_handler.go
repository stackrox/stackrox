package compliance

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/roxmetadata"
)

// CommandHandler executes the input scrape commands, and reconciles scrapes with input ComplianceReturns,
// outputing the ScrapeUpdates we expect to be sent back to central.
type CommandHandler interface {
	Start()
	Stop(err error)
	Stopped() concurrency.ReadOnlyErrorSignal

	SendCommand(*central.ScrapeCommand) bool
	Output() <-chan *central.ScrapeUpdate
}

// NewCommandHandler returns a new instance of a CommandHandler using the input image and Orchestrator.
func NewCommandHandler(roxMetadata roxmetadata.Metadata, complianceService Service) CommandHandler {
	return &commandHandlerImpl{
		roxMetadata: roxMetadata,
		service:     complianceService,

		commands: make(chan *central.ScrapeCommand),
		updates:  make(chan *central.ScrapeUpdate),

		scrapeIDToState: make(map[string]*scrapeState),

		stopC:    concurrency.NewErrorSignal(),
		stoppedC: concurrency.NewErrorSignal(),
	}
}
