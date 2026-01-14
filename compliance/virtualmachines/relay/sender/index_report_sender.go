// Package sender handles sending index reports to sensor and retrying on errors
package sender

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/metrics"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/logging"
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

	start := time.Now()
	sendToSensorCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req := &sensor.UpsertVirtualMachineIndexReportRequest{
		IndexReport:    indexReport,
		DiscoveredData: vmReport.GetDiscoveredData(),
	}

	resp, err := s.sensorClient.UpsertVirtualMachineIndexReport(sendToSensorCtx, req)

	if resp != nil && !resp.GetSuccess() && err == nil {
		log.Errorf("Sending index report didn't return an error but response indicated failure: %v", resp)
		err = status.Error(codes.Internal, "sensor failed to handle virtual machine index report")
	}

	result := "success"
	if err != nil {
		result = "retry"
	}
	duration := time.Since(start).Seconds()
	metrics.VMIndexReportSendAttempts.With(prometheus.Labels{"result": result}).Inc()
	metrics.VMIndexReportSendDurationSeconds.With(prometheus.Labels{"result": result}).Observe(duration)

	metrics.IndexReportsSentToSensor.With(prometheus.Labels{"failed": strconv.FormatBool(err != nil)}).Inc()

	return err
}
