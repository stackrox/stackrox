package index

import (
	"context"
	"maps"
	"strconv"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/virtualmachine"
	"github.com/stackrox/rox/sensor/common/virtualmachine/metrics"
	"google.golang.org/grpc"
)

var log = logging.LoggerForModule()

const indexReportSendTimeout = 10 * time.Second

type serviceImpl struct {
	sensor.UnimplementedVirtualMachineIndexReportServiceServer
	handler Handler
	store   VirtualMachineStore
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
	cid, err := strconv.ParseUint(ir.GetVsockCid(), 10, 32)
	if err != nil {
		return &sensor.UpsertVirtualMachineIndexReportResponse{
			Success: false,
		}, errox.InvalidArgs.CausedBy(errors.Wrapf(err, "invalid vsock CID: %q", ir.GetVsockCid()))
	}

	log.Debugf("Upserting virtual machine index report with vsock_cid=%q", ir.GetVsockCid())

	data := req.GetDiscoveredData()
	// Store discovered facts if feature is enabled and we have discovered data
	if features.VirtualMachines.Enabled() && data != nil {
		if s.store != nil {
			vmInfo := s.store.GetFromCID(uint32(cid))
			if vmInfo == nil {
				log.Debugf("VM with vsock_cid=%q not found, skipping discovered facts storage", ir.GetVsockCid())
				metrics.IndexReportsForUnknownVMCID.Inc()
			} else if err := s.storeDiscoveredFacts(ctx, vmInfo.ID, data); err != nil {
				log.Warnf("Failed to store discovered facts for vm_id=%q: %v", vmInfo.ID, err)
			}
		}
		detectedOS := data.GetDetectedOs()
		osVersion := data.GetOsVersion()
		activationStatus := data.GetActivationStatus()
		dnfMetadataStatus := data.GetDnfMetadataStatus()
		log.Debugf("VM discovered data: detectedOS=%s, osVersion=%q, activationStatus=%s, dnfMetadataStatus=%s",
			detectedOS.String(), osVersion, activationStatus.String(), dnfMetadataStatus.String())
		// Record metric for VM discovered data for customer data debugging purposes.
		metrics.VMDiscoveredData.With(prometheus.Labels{
			"detected_os":         detectedOS.String(),
			"activation_status":   activationStatus.String(),
			"dnf_metadata_status": dnfMetadataStatus.String(),
		}).Inc()
	}

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

// storeDiscoveredFacts converts DiscoveredData to a map[string]string and stores it by VM ID.
func (s *serviceImpl) storeDiscoveredFacts(ctx context.Context, vmID virtualmachine.VMID, data *v1.DiscoveredData) error {
	if data == nil {
		return nil
	}

	// Convert DiscoveredData to map[string]string with machine-readable keys
	facts := factsFromDiscoveredData(data)
	if len(facts) == 0 {
		return nil
	}
	previousFacts := s.store.GetDiscoveredFacts(vmID)
	s.store.UpsertDiscoveredFacts(vmID, facts)
	if !maps.Equal(previousFacts, facts) {
		if err := s.handler.SendVirtualMachineUpdate(ctx, vmID); err != nil {
			log.Warnf("Failed to emit virtual machine update after discovered facts upsert for vm_id=%q: %v", vmID, err)
		}
	}

	return nil
}

func factsFromDiscoveredData(data *v1.DiscoveredData) map[string]string {
	facts := make(map[string]string)
	if data.GetDetectedOs() != v1.DetectedOS_UNKNOWN {
		facts[virtualmachine.FactsDetectedOSKey] = data.GetDetectedOs().String()
	}
	if data.GetOsVersion() != "" {
		facts[virtualmachine.FactsOSVersionKey] = data.GetOsVersion()
	}
	if data.GetActivationStatus() != v1.ActivationStatus_ACTIVATION_UNSPECIFIED {
		facts[virtualmachine.FactsActivationStatusKey] = data.GetActivationStatus().String()
	}
	if data.GetDnfMetadataStatus() != v1.DnfMetadataStatus_DNF_METADATA_UNSPECIFIED {
		facts[virtualmachine.FactsDNFMetadataStatusKey] = data.GetDnfMetadataStatus().String()
	}
	return facts
}
