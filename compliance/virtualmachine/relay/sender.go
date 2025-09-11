package relay

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
)

// ReportSender provides functionality to send index reports to sensor
type ReportSender interface {
	// SendIndexReport sends a virtual machine index report to sensor
	SendIndexReport(ctx context.Context, report *v1.IndexReport) error
}

// reportSenderImpl implements ReportSender
type reportSenderImpl struct {
	client sensor.VirtualMachineIndexReportServiceClient
}

// NewReportSender creates a new report sender
func NewReportSender(client sensor.VirtualMachineIndexReportServiceClient) ReportSender {
	return &reportSenderImpl{
		client: client,
	}
}

// SendIndexReport sends a virtual machine index report to sensor
func (s *reportSenderImpl) SendIndexReport(ctx context.Context, report *v1.IndexReport) error {
	log.Infof("Relaying VM index report for vsock_cid=%s to sensor", report.VsockCid)

	// Create the request
	req := &sensor.UpsertVirtualMachineIndexReportRequest{
		IndexReport: report,
	}

	// Send the request using the existing service
	response, err := s.client.UpsertVirtualMachineIndexReport(ctx, req)
	if err != nil {
		log.Errorf("Failed to send VM index report for vsock_cid=%s: %v", report.VsockCid, err)
		return err
	}

	if !response.Success {
		log.Errorf("VM index report was not successful for vsock_cid=%s", report.VsockCid)
		return err
	}

	log.Infof("Successfully relayed VM index report for vsock_cid=%s", report.VsockCid)

	return nil
}
