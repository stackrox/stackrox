package index

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
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
	// This is temporary. In a followup, the data will be passed to Central instead of being logged.
	data := req.GetDiscoveredData()
	detectedOS := data.GetDetectedOs()
	osVersion := data.GetOsVersion()
	activationStatus := data.GetActivationStatus()
	dnfMetadataStatus := data.GetDnfMetadataStatus()
	dnfStatus := formatDnfStatusFlags(data.GetDnfStatus())
	log.Infof("VM discovered data: detected_os=%s, os_version=%q, activation_status=%s, dnf_status=[%s]",
		detectedOS.String(), osVersion, activationStatus.String(), dnfStatus)

	// Record metric for VM discovered data for customer data debugging purposes.
	metrics.VMDiscoveredData.With(prometheus.Labels{
		"detected_os":         detectedOS.String(),
		"activation_status":   activationStatus.String(),
		"dnf_metadata_status": dnfMetadataStatus.String(),
	}).Inc()
	recordDnfStatusMetrics(data.GetDnfStatus())

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

func formatDnfStatusFlags(flags []v1.DnfStatusFlag) string {
	if len(flags) == 0 {
		return "none"
	}
	names := make([]string, 0, len(flags))
	for _, f := range flags {
		names = append(names, f.String())
	}
	slices.Sort(names)
	return strings.Join(names, ", ")
}

func recordDnfStatusMetrics(flags []v1.DnfStatusFlag) {
	if len(flags) == 0 {
		metrics.VMDiscoveredDataDNFStatus.WithLabelValues("none").Inc()
		return
	}
	for _, f := range flags {
		metrics.VMDiscoveredDataDNFStatus.WithLabelValues(f.String()).Inc()
	}
}
