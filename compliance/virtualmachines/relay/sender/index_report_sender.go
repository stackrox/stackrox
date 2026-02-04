// Package sender handles sending index reports to sensor and retrying on errors
package sender

import (
	"context"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/metrics"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/retry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var log = logging.LoggerForModule()

// IndexReportSender sends index reports to Sensor.
type IndexReportSender interface {
	Send(ctx context.Context, vmReport *v1.VMReport) error
}

type sensorIndexReportSender struct {
	sensorClient sensor.VirtualMachineIndexReportServiceClient
}

var _ IndexReportSender = (*sensorIndexReportSender)(nil)

// New creates an IndexReportSender that sends reports to Sensor with retry logic.
func New(sensorClient sensor.VirtualMachineIndexReportServiceClient) IndexReportSender {
	return &sensorIndexReportSender{
		sensorClient: sensorClient,
	}
}

// Send sends the VM report to Sensor, retrying on transient errors.
func (s *sensorIndexReportSender) Send(ctx context.Context, vmReport *v1.VMReport) error {
	indexReport := vmReport.GetIndexReport()
	if indexReport == nil {
		return errors.New("VM report missing required index_report field")
	}

	log.Infof("Sending VM report to sensor (vsockCID: %s)", indexReport.GetVsockCid())

	// This is the sending logic that will be retried if needed
	sendFunc := func() error {
		sendToSensorCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		req := &sensor.UpsertVirtualMachineIndexReportRequest{
			IndexReport:    indexReport,
			DiscoveredData: vmReport.GetDiscoveredData(),
		}

		resp, err := s.sensorClient.UpsertVirtualMachineIndexReport(sendToSensorCtx, req)

		if resp != nil && !resp.GetSuccess() {
			// This can't happen as of this writing (Success is only false when an error is returned) but is
			// theoretically possible, let's add retries too.
			if err == nil {
				log.Errorf("Sending index report didn't return an error but response indicated failure: %v", resp)
				err = retry.MakeRetryable(errors.New("sensor failed to handle virtual machine index report"))
			}
		}

		if isRetryableGRPCError(err) {
			err = retry.MakeRetryable(err)
		}

		return err
	}

	onFailedAttemptsFunc := func(e error) {
		log.Warnf("Error sending index report to sensor, retrying. Error was: %v", e)
	}

	tries := 10 // With default backoff logic in pkg/retry, this takes around 50 s (without considering timeouts)

	// Considering a timeout of 5 seconds and 10 tries with exponential backoff, the maximum time until running out of
	// tries is around 1 min 40 s. Given that each virtual machine sends an index report every 4 hours, these retries
	// seem reasonable and are unlikely to cause issues.
	err := retry.WithRetry(
		sendFunc,
		retry.WithContext(ctx),
		retry.OnFailedAttempts(onFailedAttemptsFunc),
		retry.Tries(tries),
		retry.OnlyRetryableErrors(),
		retry.WithExponentialBackoff())

	metrics.IndexReportsSentToSensor.With(prometheus.Labels{"failed": strconv.FormatBool(err != nil)}).Inc()

	return err
}

func isRetryableGRPCError(err error) bool {
	grpcErr, ok := status.FromError(err)
	if !ok {
		return false
	}
	code := grpcErr.Code()
	switch code {
	case codes.DeadlineExceeded:
		return !errors.Is(err, context.Canceled)
	case codes.Unavailable, codes.ResourceExhausted, codes.Internal:
		return true
	default:
		return false
	}
}
