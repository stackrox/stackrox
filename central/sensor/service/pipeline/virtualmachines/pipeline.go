package virtualmachines

import (
	"context"
	"github.com/stackrox/rox/central/convert/internaltostorage"

	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	virtualMachineDataStore "github.com/stackrox/rox/central/virtualmachine/datastore"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/set"
)

var (
	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

func GetPipeline() pipeline.Fragment {
	return newPipeline(clusterDataStore.Singleton(), virtualMachineDataStore.Singleton())
}

func newPipeline(
	clusterStore clusterDataStore.DataStore,
	virtualMachineStore virtualMachineDataStore.DataStore,
) pipeline.Fragment {
	return &pipelineImpl{
		clusterStore:        clusterStore,
		virtualMachineStore: virtualMachineStore,
	}
}

type pipelineImpl struct {
	clusterStore        clusterDataStore.DataStore
	virtualMachineStore virtualMachineDataStore.DataStore
}

func (p *pipelineImpl) OnFinish(_ string) {}

func (p *pipelineImpl) Capabilities() []centralsensor.CentralCapability { return nil }

func (p *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetVirtualMachine() != nil
}

func (p *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	virtualMachines, err := p.virtualMachineStore.GetAllVirtualMachines(ctx)
	if err != nil {
		return errors.Wrap(err, "retrieving virtual machines for reconciliation")
	}
	clusterVMIDs := set.NewStringSet()
	for _, vm := range virtualMachines {
		if vm.GetClusterId() == clusterID {
			clusterVMIDs.Add(vm.GetId())
		}
	}

	store := storeMap.Get((*central.SensorEvent_VirtualMachine)(nil))
	return reconciliation.Perform(store, clusterVMIDs, "virtualmachines", func(id string) error {
		return p.processRemove(ctx, id)
	})
}

func (p *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.VirtualMachine)

	event := msg.GetEvent()
	virtualMachine := event.GetVirtualMachine()
	if virtualMachine == nil {
		return errors.Errorf("unexpected resource type %T for virtual machine", event.GetResource())
	}

	if event.GetAction() == central.ResourceAction_REMOVE_RESOURCE {
		return p.processRemove(ctx, virtualMachine.GetId())
	}

	virtualMachineToStore := internaltostorage.VirtualMachine(virtualMachine)

	clusterName, ok, err := p.clusterStore.GetClusterName(ctx, clusterID)
	if err == nil && ok {
		virtualMachineToStore.ClusterName = clusterName
	}

	return p.virtualMachineStore.UpsertVirtualMachine(ctx, virtualMachineToStore)
}

func (p *pipelineImpl) processRemove(ctx context.Context, id string) error {
	return p.virtualMachineStore.DeleteVirtualMachines(ctx, id)
}
