package virtualmachineindex

import (
	"context"
	"strconv"

	"github.com/pkg/errors"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	vmDatastore "github.com/stackrox/rox/central/virtualmachine/datastore"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
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
	)
}

// newPipeline returns a new instance of Pipeline.

func newPipeline(vms vmDatastore.DataStore, enricher vmEnricher.VirtualMachineEnricher) pipeline.Fragment {
	return &pipelineImpl{
		vmDatastore: vms,
		enricher:    enricher,
	}
}

type pipelineImpl struct {
	vmDatastore vmDatastore.DataStore
	enricher    vmEnricher.VirtualMachineEnricher
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

	// Parse vsock CID from index report
	vsockCidStr := index.GetIndex().GetVsockCid()
	vsockCid, err := strconv.ParseInt(vsockCidStr, 10, 32)
	if err != nil {
		return errors.Wrapf(err, "invalid vsock CID in index report: %q", vsockCidStr)
	}

	// Get or create VM
	vm := &storage.VirtualMachine{
		Id:       index.GetId(),
		VsockCid: int32(vsockCid),
	}

	// Extract Scanner V4 index report from VM index report event
	indexV4 := index.GetIndex().GetIndexV4()
	if indexV4 == nil {
		return errors.Errorf("VM index report %s missing Scanner V4 index data", index.GetId())
	}

	// Enrich VM with vulnerabilities
	err = p.enricher.EnrichVirtualMachineWithVulnerabilities(vm, indexV4)
	if err != nil {
		return errors.Wrapf(err, "failed to enrich VM %s with vulnerabilities", index.GetId())
	}

	// Store enriched VM
	if err := p.vmDatastore.UpdateVirtualMachineScan(ctx, vm.GetId(), vm.GetScan()); err != nil {
		// If VM doesn't exist and test mode is enabled, auto-create it
		if errors.Is(err, errox.NotFound) && env.IsVMTestModeEnabled() {
			log.Debugf("VM %s not found in database - auto-creating VM record with scan data (test mode enabled)", vm.GetId())

			// Populate VM metadata from index report for test mode
			vm.Name = "vm-" + vsockCidStr
			vm.Namespace = "vm-load-test"
			vm.State = storage.VirtualMachine_RUNNING

			if upsertErr := p.vmDatastore.UpsertVirtualMachine(ctx, vm); upsertErr != nil {
				return errors.Wrapf(upsertErr, "failed to create VM %s in datastore", index.GetId())
			}
			log.Debugf("Successfully auto-created VM %s (name: %s) with %d components from index report",
				vm.GetId(), vm.GetName(), len(vm.GetScan().GetComponents()))
		} else {
			return errors.Wrapf(err, "failed to update VM %s scan in datastore", index.GetId())
		}
	} else {
		log.Debugf("Successfully enriched and stored VM %s with %d components",
			vm.GetId(), len(vm.GetScan().GetComponents()))
	}

	return nil
}
