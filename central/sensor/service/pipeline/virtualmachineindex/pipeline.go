package virtualmachineindex

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/convert/v1tov2storage"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	virtualMachineDataStore "github.com/stackrox/rox/central/virtualmachine/datastore"
	virtualMachineV2DataStore "github.com/stackrox/rox/central/virtualmachine/v2/datastore"
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
		virtualMachineDataStore.Singleton(),
		vmEnricher.Singleton(),
		virtualMachineV2DataStore.Singleton(),
	)
}

// newPipeline returns a new instance of Pipeline.
func newPipeline(
	virtualMachineStore virtualMachineDataStore.DataStore,
	enricher vmEnricher.VirtualMachineEnricher,
	virtualMachineV2Store virtualMachineV2DataStore.DataStore,
) pipeline.Fragment {
	return &pipelineImpl{
		virtualMachineStore:   virtualMachineStore,
		enricher:              enricher,
		virtualMachineV2Store: virtualMachineV2Store,
	}
}

type pipelineImpl struct {
	virtualMachineStore   virtualMachineDataStore.DataStore
	enricher              vmEnricher.VirtualMachineEnricher
	virtualMachineV2Store virtualMachineV2DataStore.DataStore
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

	event := msg.GetEvent()
	index := event.GetVirtualMachineIndexReport()
	resourceID := ""
	if index != nil {
		resourceID = common.VMIndexACKResourceID(index.GetId(), index.GetIndex().GetVsockCid())
	}

	if !features.VirtualMachines.Enabled() {
		// ACK to prevent the sender from retrying when the feature is disabled on Central.
		sendVMIndexACK(ctx, resourceID, centralsensor.SensorACKReasonFeatureDisabled, injector)
		return nil
	}
	if index == nil {
		return errors.Errorf("unexpected resource type %T for virtual machine index report", event.GetResource())
	}
	if event.GetAction() != central.ResourceAction_SYNC_RESOURCE {
		log.Warnf(
			"Action %s on virtual machine index reports is not supported. Only %s is supported.",
			event.GetAction().String(),
			central.ResourceAction_SYNC_RESOURCE.String(),
		)
		sendVMIndexNACK(ctx, resourceID, centralsensor.SensorACKReasonUnsupportedAction, injector)
		return nil
	}

	log.Debugf("Received virtual machine index report: %s", index.GetId())

	if clusterID == "" {
		sendVMIndexNACK(ctx, resourceID, centralsensor.SensorACKReasonMissingClusterID, injector)
		return errors.New("missing cluster ID in pipeline context")
	}

	// Get or create VM
	vm := &storage.VirtualMachine{Id: index.GetId()}

	// Extract Scanner V4 index report from VM index report event
	indexV4 := index.GetIndex().GetIndexV4()
	if indexV4 == nil {
		sendVMIndexNACK(ctx, resourceID, centralsensor.SensorACKReasonMissingScanData, injector)
		return errors.Errorf("VM index report %s missing Scanner V4 index data", index.GetId())
	}

	// Enrich VM with vulnerabilities
	err := p.enricher.EnrichVirtualMachineWithVulnerabilities(vm, indexV4)
	if err != nil {
		sendVMIndexNACK(ctx, resourceID, centralsensor.SensorACKReasonEnrichmentFailed, injector)
		return errors.Wrapf(err, "failed to enrich VM %s with vulnerabilities", index.GetId())
	}

	// Store enriched VM via v1 or v2 path.
	if features.VirtualMachinesEnhancedDataModel.Enabled() {
		if err := p.storeV2Scan(ctx, clusterID, vm); err != nil {
			sendVMIndexNACK(ctx, resourceID, centralsensor.SensorACKReasonStorageFailed, injector)
			return err
		}
	} else {
		if err := p.storeV1Scan(ctx, vm); err != nil {
			sendVMIndexNACK(ctx, resourceID, centralsensor.SensorACKReasonStorageFailed, injector)
			return err
		}
	}

	log.Debugf("Successfully enriched and stored VM %s with %d components",
		vm.GetId(), len(vm.GetScan().GetComponents()))

	sendVMIndexACK(ctx, resourceID, "", injector)
	return nil
}

func (p *pipelineImpl) storeV1Scan(ctx context.Context, vm *storage.VirtualMachine) error {
	if err := p.virtualMachineStore.UpdateVirtualMachineScan(ctx, vm.GetId(), vm.GetScan()); err != nil {
		return errors.Wrapf(err, "failed to upsert VM %s to datastore", vm.GetId())
	}
	return nil
}

func (p *pipelineImpl) storeV2Scan(ctx context.Context, clusterID string, vm *storage.VirtualMachine) error {
	if err := p.virtualMachineV2Store.EnsureVirtualMachineExists(ctx, vm.GetId(), clusterID); err != nil {
		return errors.Wrapf(err, "failed to ensure VM %s exists in v2 datastore", vm.GetId())
	}

	parts := v1tov2storage.ScanPartsFromV1Scan(vm.GetId(), vm.GetScan())
	if parts == nil {
		return nil
	}

	if err := p.virtualMachineV2Store.UpsertScan(ctx, vm.GetId(), *parts); err != nil {
		return errors.Wrapf(err, "failed to upsert v2 scan for VM %s", vm.GetId())
	}
	return nil
}
