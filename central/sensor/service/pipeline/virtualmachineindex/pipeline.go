package virtualmachineindex

import (
	"context"
	"strconv"

	"github.com/pkg/errors"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	vmDatastore "github.com/stackrox/rox/central/virtualmachine/datastore"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/rate"
	vmEnricher "github.com/stackrox/rox/pkg/virtualmachine/enricher"
)

const (
	// rateLimiterWorkload is the workload name used for rate limiting VM index reports.
	rateLimiterWorkload = "vm_index_report"
)

// rateLimiter defines the interface for rate limiting operations used by this pipeline.
// This interface is satisfied by *rate.Limiter and allows for easier testing.
type rateLimiter interface {
	TryConsume(clientID string) (allowed bool, reason string)
	OnClientDisconnect(clientID string)
}

var (
	log = logging.LoggerForModule()

	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	rateLimit, err := strconv.ParseFloat(env.VMIndexReportRateLimit.Setting(), 64)
	if err != nil {
		log.Warnf("Invalid %s value: %v. Using fallback value of 1.0", env.VMIndexReportRateLimit.EnvVar(), err)
		rateLimit = 1.0
	}
	bucketCapacity := env.VMIndexReportBucketCapacity.IntegerSetting()
	if bucketCapacity < 1 {
		log.Warnf("Invalid %s value: %d. Using fallback value of 1.", env.VMIndexReportBucketCapacity.EnvVar(), bucketCapacity)
		bucketCapacity = 1
	}
	rateLimiter, err := rate.NewLimiter(rateLimiterWorkload, rateLimit, bucketCapacity)
	if err != nil {
		log.Panicf("Failed to create rate limiter for %s: %v", rateLimiterWorkload, err)
	}
	return newPipeline(
		vmDatastore.Singleton(),
		vmEnricher.Singleton(),
		rateLimiter,
	)
}

// newPipeline returns a new instance of Pipeline.
func newPipeline(vms vmDatastore.DataStore, enricher vmEnricher.VirtualMachineEnricher, rl rateLimiter) pipeline.Fragment {
	return &pipelineImpl{
		vmDatastore: vms,
		enricher:    enricher,
		rateLimiter: rl,
	}
}

type pipelineImpl struct {
	vmDatastore vmDatastore.DataStore
	enricher    vmEnricher.VirtualMachineEnricher
	rateLimiter rateLimiter
}

func (p *pipelineImpl) OnFinish(clusterID string) {
	// Notify rate limiter that this client (Sensor) has disconnected so it can rebalance the limiters.
	if p.rateLimiter != nil {
		p.rateLimiter.OnClientDisconnect(clusterID)
	}
}

func (p *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return []centralsensor.CentralCapability{centralsensor.VirtualMachinesSupported}
}

func (p *pipelineImpl) Reconcile(_ context.Context, _ string, _ *reconciliation.StoreMap) error {
	return nil
}

func (p *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetVirtualMachineIndexReport() != nil
}

func (p *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, injector common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.VirtualMachineIndex)

	if !features.VirtualMachines.Enabled() {
		return nil
	}
	event := msg.GetEvent()
	index := event.GetVirtualMachineIndexReport()
	if index == nil {
		return errors.Errorf("unexpected resource type %T for virtual machine index report", event.GetResource())
	}
	if event.GetAction() != central.ResourceAction_SYNC_RESOURCE {
		log.Warnf(
			"Action %s on virtual machine index reports is not supported. Only %s is supported.",
			event.GetAction().String(),
			central.ResourceAction_SYNC_RESOURCE.String(),
		)
		return nil
	}

	log.Debugf("Received virtual machine index report: %s", index.GetId())

	if clusterID == "" {
		return errors.New("missing cluster ID in pipeline context")
	}

	// Extract connection for capability checks; cluster ID is taken from the pipeline argument.
	conn := connection.FromContext(ctx)

	// Rate limit check. Drop message if rate limiter is misconfigured (defensive behavior against misconfiguration)
	// or rate limit exceeded. Afterwards, send NACK to Sensor if Sensor supports it.
	if p.rateLimiter == nil {
		logging.GetRateLimitedLogger().WarnL(
			"vm_index_report_nil_rate_limiter",
			"No rate limiter found for workload %q. Dropping VM index report from cluster %s",
			rateLimiterWorkload,
			clusterID,
		)
		if conn != nil && conn.HasCapability(centralsensor.SensorACKSupport) {
			sendVMIndexReportResponse(ctx, clusterID, index.GetId(), central.SensorACK_NACK, "rate limiter not configured", injector)
		}
		return nil // Don't return error - would cause pipeline retry
	}

	allowed, reason := p.rateLimiter.TryConsume(clusterID)
	if !allowed {
		log.Infof("Dropping VM index report %s from cluster %s: %s", index.GetId(), clusterID, reason)
		if conn != nil && conn.HasCapability(centralsensor.SensorACKSupport) {
			sendVMIndexReportResponse(ctx, clusterID, index.GetId(), central.SensorACK_NACK, reason, injector)
		}
		return nil // Don't return error - would cause pipeline retry
	}

	// Get or create VM
	vm := &storage.VirtualMachine{Id: index.GetId()}

	// Extract Scanner V4 index report from VM index report event
	indexV4 := index.GetIndex().GetIndexV4()
	if indexV4 == nil {
		return errors.Errorf("VM index report %s missing Scanner V4 index data", index.GetId())
	}

	// Enrich VM with vulnerabilities
	err := p.enricher.EnrichVirtualMachineWithVulnerabilities(vm, indexV4)
	if err != nil {
		return errors.Wrapf(err, "failed to enrich VM %s with vulnerabilities", index.GetId())
	}

	// Store enriched VM
	if err := p.vmDatastore.UpdateVirtualMachineScan(ctx, vm.GetId(), vm.GetScan()); err != nil {
		return errors.Wrapf(err, "failed to upsert VM %s to datastore", index.GetId())
	}

	log.Debugf("Successfully enriched and stored VM %s with %d components",
		vm.GetId(), len(vm.GetScan().GetComponents()))

	// Send ACK to Sensor if Sensor supports it
	if conn != nil && conn.HasCapability(centralsensor.SensorACKSupport) {
		sendVMIndexReportResponse(ctx, clusterID, index.GetId(), central.SensorACK_ACK, "", injector)
	}
	return nil
}

// sendVMIndexReportResponse sends an ACK or NACK for a VM index report.
func sendVMIndexReportResponse(ctx context.Context, clusterID, vmID string, action central.SensorACK_Action, reason string, injector common.MessageInjector) {
	if injector == nil {
		log.Debugf("Cannot send %s to Sensor for cluster %s - no injector", action.String(), clusterID)
		return
	}
	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_SensorAck{
			SensorAck: &central.SensorACK{
				Action:      action,
				MessageType: central.SensorACK_VM_INDEX_REPORT,
				ResourceId:  vmID,
				Reason:      reason,
			},
		},
	}
	if err := injector.InjectMessage(ctx, msg); err != nil {
		log.Warnf("Failed sending VM index report %s for VM %s in cluster %s: %v", action.String(), vmID, clusterID, err)
	} else {
		log.Debugf("Sent VM index report %s for VM %s in cluster %s (reason=%q)", action.String(), vmID, clusterID, reason)
	}
}
