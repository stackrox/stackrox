package virtualmachineindex

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/convert/v1tov2storage"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	vmDatastore "github.com/stackrox/rox/central/virtualmachine/datastore"
	vmV2DataStore "github.com/stackrox/rox/central/virtualmachine/v2/datastore"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	vmEnricher "github.com/stackrox/rox/pkg/virtualmachine/enricher"
)

var (
	log = logging.LoggerForModule()

	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return newPipeline(
		vmDatastore.Singleton(),
		vmEnricher.Singleton(),
		vmV2DataStore.Singleton(),
	)
}

// newPipeline returns a new instance of Pipeline.
func newPipeline(vms vmDatastore.DataStore, enricher vmEnricher.VirtualMachineEnricher, vmV2Store vmV2DataStore.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		vmDatastore: vms,
		enricher:    enricher,
		vmV2Store:   vmV2Store,
	}
}

type pipelineImpl struct {
	vmDatastore vmDatastore.DataStore
	enricher    vmEnricher.VirtualMachineEnricher
	vmV2Store   vmV2DataStore.DataStore
}

func (p *pipelineImpl) OnFinish(string) {}

func (p *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return []centralsensor.CentralCapability{centralsensor.VirtualMachinesSupported}
}

func (p *pipelineImpl) Reconcile(_ context.Context, _ string, _ *reconciliation.StoreMap) error {
	return nil
}

func (p *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetVirtualMachineIndexReport() != nil
}

// sendVMIndexACK is a convenience wrapper around common.SendSensorACK for VM index reports.
func sendVMIndexACK(ctx context.Context, resourceID, reason string, injector common.MessageInjector) {
	common.SendSensorACK(ctx, central.SensorACK_ACK, central.SensorACK_VM_INDEX_REPORT, resourceID, reason, injector)
}

// sendVMIndexNACK is a convenience wrapper around common.SendSensorACK for VM index reports.
func sendVMIndexNACK(ctx context.Context, resourceID, reason string, injector common.MessageInjector) {
	common.SendSensorACK(ctx, central.SensorACK_NACK, central.SensorACK_VM_INDEX_REPORT, resourceID, reason, injector)
}

func (p *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, injector common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.VirtualMachineIndex)

	if !features.VirtualMachines.Enabled() {
		// ACK to prevent the sender from retrying when the feature is disabled on Central.
		sendVMIndexACK(ctx, msg.GetEvent().GetVirtualMachineIndexReport().GetId(), centralsensor.SensorACKReasonFeatureDisabled, injector)
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
		sendVMIndexNACK(ctx, index.GetId(), centralsensor.SensorACKReasonUnsupportedAction, injector)
		return nil
	}

	log.Debugf("Received virtual machine index report: %s", index.GetId())

	if clusterID == "" {
		sendVMIndexNACK(ctx, index.GetId(), centralsensor.SensorACKReasonMissingClusterID, injector)
		return errors.New("missing cluster ID in pipeline context")
	}

	// Get or create VM
	vm := &storage.VirtualMachine{Id: index.GetId()}

	// Extract Scanner V4 index report from VM index report event
	indexV4 := index.GetIndex().GetIndexV4()
	if indexV4 == nil {
		sendVMIndexNACK(ctx, index.GetId(), centralsensor.SensorACKReasonMissingScanData, injector)
		return errors.Errorf("VM index report %s missing Scanner V4 index data", index.GetId())
	}

	// Enrich VM with vulnerabilities
	err := p.enricher.EnrichVirtualMachineWithVulnerabilities(vm, indexV4)
	if err != nil {
		sendVMIndexNACK(ctx, index.GetId(), centralsensor.SensorACKReasonEnrichmentFailed, injector)
		return errors.Wrapf(err, "failed to enrich VM %s with vulnerabilities", index.GetId())
	}

	if p.vmV2Store != nil {
		// Upsert minimal VM record to satisfy FK constraint.
		vmV2 := &storage.VirtualMachineV2{Id: vm.GetId(), ClusterId: clusterID}
		if err := p.vmV2Store.UpsertVirtualMachine(ctx, vmV2); err != nil {
			return errors.Wrapf(err, "failed to upsert VM v2 %s to datastore", index.GetId())
		}
		// Convert v1 scan to v2 parts and upsert.
		scanParts := v1tov2storage.ScanPartsFromV1Scan(vm.GetId(), vm.GetScan())
		if err := p.vmV2Store.UpsertScan(ctx, vm.GetId(), scanParts); err != nil {
			return errors.Wrapf(err, "failed to upsert VM v2 scan %s to datastore", index.GetId())
		}
	} else {
		if err := p.vmDatastore.UpdateVirtualMachineScan(ctx, vm.GetId(), vm.GetScan()); err != nil {
			return errors.Wrapf(err, "failed to upsert VM %s to datastore", index.GetId())
		}
	}

	log.Debugf("Successfully enriched and stored VM %s with %d components",
		vm.GetId(), len(vm.GetScan().GetComponents()))

	sendVMIndexACK(ctx, index.GetId(), "", injector)
	return nil
}
