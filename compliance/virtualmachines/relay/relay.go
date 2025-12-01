package relay

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/provider"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/sender"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

type Relay struct {
	reportProvider provider.ReportProvider
	reportSender   sender.ReportSender
}

// NewRelay creates a Relay with the given provider and sender.
func NewRelay(reportProvider provider.ReportProvider, reportSender sender.ReportSender) *Relay {
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
