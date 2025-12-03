package relay

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/sender"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

// IndexReportStream manages report collection and produces validated reports.
type IndexReportStream interface {
	// Start begins accepting connections and returns a channel of validated reports.
	// The channel is currently not closed to avoid races during shutdown.
	// TODO: Implement proper shutdown logic that closes the channel.
	Start(ctx context.Context) (<-chan *v1.IndexReport, error)
}

type Relay struct {
	reportStream IndexReportStream
	reportSender sender.IndexReportSender
}

// New creates a Relay with the given report stream and sender.
func New(reportStream IndexReportStream, reportSender sender.IndexReportSender) *Relay {
	return &Relay{
		reportStream: reportStream,
		reportSender: reportSender,
	}
}

func (r *Relay) Run(ctx context.Context) error {
	log.Info("Starting virtual machine relay")

	reportChan, err := r.reportStream.Start(ctx)
	if err != nil {
		return errors.Wrap(err, "starting report stream")
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
