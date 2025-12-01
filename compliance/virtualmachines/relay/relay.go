package relay

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/provider"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/sender"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
)

var log = logging.LoggerForModule()

type Relay struct {
	reportProvider provider.ReportProvider
	reportSender   sender.ReportSender
}

func NewRelay(conn grpc.ClientConnInterface) (*Relay, error) {
	sensorClient := sensor.NewVirtualMachineIndexReportServiceClient(conn)

	reportProvider, err := provider.New()
	if err != nil {
		return nil, errors.Wrap(err, "creating report provider")
	}

	return &Relay{
		reportProvider: reportProvider,
		reportSender:   sender.New(sensorClient),
	}, nil
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
