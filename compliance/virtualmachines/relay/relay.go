package relay

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/sender"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

// IndexReportProvider manages report collection and produces validated reports.
type IndexReportProvider interface {
	// Start begins accepting connections and returns a channel of validated reports.
	// The channel is currently not closed to avoid races during shutdown.
	// TODO: Implement proper shutdown logic that closes the channel.
	Start(ctx context.Context) (<-chan *v1.IndexReport, error)
}

type Relay struct {
	reportProvider IndexReportProvider
	reportSender   sender.ReportSender
}

// New creates a Relay with the given provider and sender.
func New(reportProvider IndexReportProvider, reportSender sender.ReportSender) *Relay {
	return &Relay{
		reportProvider: reportProvider,
		reportSender:   reportSender,
	}
}

func (r *Relay) Run(ctx context.Context) error {
	log.Info("Starting virtual machine relay")

	reportChan, err := r.reportProvider.Start(ctx)
	if err != nil {
		return errors.Wrap(err, "starting report provider")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case report := <-reportChan:
			if err := r.reportSender.Send(ctx, report); err != nil {
				log.Errorf("Failed to send report (vsock CID: %s): %v",
					report.GetVsockCid(), err)
			}
		}
	}
}
