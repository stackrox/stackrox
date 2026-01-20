package index

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	vmv1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/virtualmachine/metrics"
	"google.golang.org/grpc"
)

var log = logging.LoggerForModule()

const indexReportSendTimeout = 10 * time.Second

type serviceImpl struct {
	sensor.UnimplementedVirtualMachineIndexReportServiceServer
	handler Handler
}

var _ Service = (*serviceImpl)(nil)

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	sensor.RegisterVirtualMachineIndexReportServiceServer(grpcServer, s)
}

// RegisterServiceHandler implements the APIService interface, but the agent
// does not accept calls over the gRPC gateway.
func (s *serviceImpl) RegisterServiceHandler(context.Context, *runtime.ServeMux, *grpc.ClientConn) error {
	return nil
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	if err := idcheck.CollectorOnly().Authorized(ctx, fullMethodName); err != nil {
		return ctx, errors.Wrapf(err, "virtual machine index report authorization for %q", fullMethodName)
	}
	return ctx, nil
}

func (s *serviceImpl) UpsertVirtualMachineIndexReport(ctx context.Context, req *sensor.UpsertVirtualMachineIndexReportRequest) (*sensor.UpsertVirtualMachineIndexReportResponse, error) {
	startTime := time.Now()
	defer func() {
		metrics.VirtualMachineIndexReportHandlingDurationMilliseconds.
			Observe(metrics.StartTimeToMS(startTime))
	}()

	ir := req.GetIndexReport()
	if ir == nil {
		return &sensor.UpsertVirtualMachineIndexReportResponse{
			Success: false,
		}, errox.InvalidArgs.CausedBy("index report in request cannot be nil")
	}

	log.Debugf("Upserting virtual machine index report with vsock_cid=%q", ir.GetVsockCid())

	// Log VM discovered data.
	// TODO: This is temporary. In a followup, logging will be reduced to Debug level
	// and sanitized to avoid potential sensitive data leakage.
	discoveredData := req.GetDiscoveredData()
	detectedOS := ""
	activationStatus := vmv1.ActivationStatus_ACTIVATION_STATUS_UNSPECIFIED
	dnfMetadataStatus := vmv1.DnfMetadataStatus_DNF_METADATA_STATUS_UNSPECIFIED
	if discoveredData != nil {
		detectedOS = discoveredData.GetDetectedOs()
		activationStatus = discoveredData.GetActivationStatus()
		dnfMetadataStatus = discoveredData.GetDnfMetadataStatus()
	}
	log.Infof("VM discovered data: detected_os=%q, activation_status=%s, dnf_metadata_status=%s",
		detectedOS, activationStatus.String(), dnfMetadataStatus.String())

	// Record metric for VM discovered data with all labels.
	// TODO: This is temporary. In a followup, detected_os will be normalized to a small fixed set
	// of OS categories (e.g., rhel, ubuntu, debian, suse, windows, unknown) to avoid high-cardinality metrics.
	metrics.VMDiscoveredData.With(prometheus.Labels{
		"detected_os":         detectedOS,
		"activation_status":   activationStatus.String(),
		"dnf_metadata_status": dnfMetadataStatus.String(),
	}).Inc()

	metrics.IndexReportsReceived.Inc()
	timeoutCtx, cancel := context.WithTimeout(ctx, indexReportSendTimeout)
	defer cancel()
	if err := s.handler.Send(timeoutCtx, ir); err != nil {
		return &sensor.UpsertVirtualMachineIndexReportResponse{
			Success: false,
		}, errors.Wrapf(err, "sending virtual machine index report with vsock_cid=%q to Central", ir.GetVsockCid())
	}
	return &sensor.UpsertVirtualMachineIndexReportResponse{
		Success: true,
	}, nil
}
