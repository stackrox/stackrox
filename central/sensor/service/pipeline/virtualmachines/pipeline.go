package virtualmachines

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/convert/internaltostorage"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	virtualMachineDataStore "github.com/stackrox/rox/central/virtualmachine/datastore"
	vmV2DataStore "github.com/stackrox/rox/central/virtualmachine/v2/datastore"
	"github.com/stackrox/rox/generated/internalapi/central"
	virtualMachineV1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

var (
	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

func GetPipeline() pipeline.Fragment {
	return newPipeline(clusterDataStore.Singleton(), virtualMachineDataStore.Singleton(), vmV2DataStore.Singleton())
}

func newPipeline(
	clusterStore clusterDataStore.DataStore,
	virtualMachineStore virtualMachineDataStore.DataStore,
	vmV2Store vmV2DataStore.DataStore,
) pipeline.Fragment {
	return &pipelineImpl{
		clusterStore:        clusterStore,
		virtualMachineStore: virtualMachineStore,
		vmV2Store:           vmV2Store,
	}
}

type pipelineImpl struct {
	clusterStore        clusterDataStore.DataStore
	virtualMachineStore virtualMachineDataStore.DataStore
	vmV2Store           vmV2DataStore.DataStore
}

func (p *pipelineImpl) OnFinish(_ string) {}

func (p *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return []centralsensor.CentralCapability{centralsensor.VirtualMachinesSupported}
}

func (p *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetVirtualMachine() != nil
}

func (p *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	if p.vmV2Store != nil {
		return p.reconcileV2(ctx, clusterID, storeMap)
	}
	return p.reconcileV1(ctx, clusterID, storeMap)
}

func (p *pipelineImpl) reconcileV1(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	virtualMachines, err := p.virtualMachineStore.SearchRawVirtualMachines(ctx, search.EmptyQuery())
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
		return p.virtualMachineStore.DeleteVirtualMachines(ctx, id)
	})
}

func (p *pipelineImpl) reconcileV2(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	virtualMachines, err := p.vmV2Store.SearchRawVirtualMachines(ctx, search.EmptyQuery())
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
		return p.vmV2Store.DeleteVirtualMachines(ctx, id)
	})
}

func (p *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.VirtualMachine)

	event := msg.GetEvent()
	virtualMachine := event.GetVirtualMachine()
	if virtualMachine == nil {
		return errors.Errorf("unexpected resource type %T for virtual machine", event.GetResource())
	}

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		if p.vmV2Store != nil {
			return p.vmV2Store.DeleteVirtualMachines(ctx, virtualMachine.GetId())
		}
		return p.virtualMachineStore.DeleteVirtualMachines(ctx, virtualMachine.GetId())
	case central.ResourceAction_CREATE_RESOURCE, central.ResourceAction_UPDATE_RESOURCE, central.ResourceAction_SYNC_RESOURCE:
		if p.vmV2Store != nil {
			return p.runUpsertPipelineV2(ctx, clusterID, virtualMachine)
		}
		return p.runUpsertPipeline(ctx, clusterID, virtualMachine)
	default:
		return fmt.Errorf("event action '%s' for virtual machine does not exist", event.GetAction())
	}
}

func (p *pipelineImpl) runUpsertPipeline(
	ctx context.Context,
	clusterID string,
	vm *virtualMachineV1.VirtualMachine,
) error {
	virtualMachineToStore := internaltostorage.VirtualMachine(vm)

	clusterName, ok, err := p.clusterStore.GetClusterName(ctx, clusterID)
	if err == nil && ok {
		virtualMachineToStore.ClusterName = clusterName
	}

	return p.virtualMachineStore.UpsertVirtualMachine(ctx, virtualMachineToStore)
}

func (p *pipelineImpl) runUpsertPipelineV2(
	ctx context.Context,
	clusterID string,
	vm *virtualMachineV1.VirtualMachine,
) error {
	vmV2 := internaltostorage.VirtualMachineV2(vm)

	clusterName, ok, err := p.clusterStore.GetClusterName(ctx, clusterID)
	if err == nil && ok {
		vmV2.ClusterName = clusterName
	}

	return p.vmV2Store.UpsertVirtualMachine(ctx, vmV2)
}
