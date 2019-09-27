package compliance

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/orchestrators"
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

	resultsChan() chan<- *compliance.ComplianceReturn

	GetScrapeConfig(ctx context.Context, nodeName, scrapeID string) (*sensor.ScrapeConfig, error)
}

// NewCommandHandler returns a new instance of a CommandHandler using the input image and Orchestrator.
func NewCommandHandler(orchestrator orchestrators.Orchestrator, roxMetadata roxmetadata.Metadata) CommandHandler {
	return &commandHandlerImpl{
		orchestrator: orchestrator,

		roxMetadata: roxMetadata,

		commands: make(chan *central.ScrapeCommand),
		updates:  make(chan *central.ScrapeUpdate),

		scrapeIDToState: make(map[string]*scrapeState),

		stopC:    concurrency.NewErrorSignal(),
		stoppedC: concurrency.NewErrorSignal(),

		resultsC: make(chan *compliance.ComplianceReturn),
	}
}
