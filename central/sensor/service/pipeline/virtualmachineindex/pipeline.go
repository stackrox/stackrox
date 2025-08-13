package virtualmachineindex

import (
	"context"

	"github.com/pkg/errors"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	vmDatastore "github.com/stackrox/rox/central/virtualmachine/datastore"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	log = logging.LoggerForModule()

	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return newPipeline(vmDatastore.Singleton())
}

// newPipeline returns a new instance of Pipeline.
func newPipeline(vms vmDatastore.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		vmDatastore: vms,
	}
}

type pipelineImpl struct {
	vmDatastore vmDatastore.DataStore
}

func (p *pipelineImpl) OnFinish(_ string) {
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

func (p *pipelineImpl) Run(ctx context.Context, _ string, msg *central.MsgFromSensor, injector common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.VirtualMachineIndex)

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

	log.Debugf("Received virtual machine index report message: %s", index.GetId())

	// TODO: This is where the virtual machine enrichment would be run. See `nodeindex` pipeline.
	// Here we just do a dummy conversion instead.
	if err := p.vmDatastore.UpsertVirtualMachine(ctx, &storage.VirtualMachine{Id: index.GetId()}); err != nil {
		return errors.Wrap(err, "failed to upsert virtual machine to datstore")
	}

	return nil
}
