package virtualmachines

import (
	"context"
	"fmt"
	"math"

	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/convert/internaltostorage"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	virtualMachineDataStore "github.com/stackrox/rox/central/virtualmachine/datastore"
	virtualMachineV2DataStore "github.com/stackrox/rox/central/virtualmachine/v2/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	virtualMachineV1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

var (
	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

func GetPipeline() pipeline.Fragment {
	return newPipeline(clusterDataStore.Singleton(), virtualMachineDataStore.Singleton(), virtualMachineV2DataStore.Singleton())
}

func newPipeline(
	clusterStore clusterDataStore.DataStore,
	virtualMachineStore virtualMachineDataStore.DataStore,
	virtualMachineV2Store virtualMachineV2DataStore.DataStore,
) pipeline.Fragment {
	return &pipelineImpl{
		clusterStore:          clusterStore,
		virtualMachineStore:   virtualMachineStore,
		virtualMachineV2Store: virtualMachineV2Store,
	}
}

type pipelineImpl struct {
	clusterStore          clusterDataStore.DataStore
	virtualMachineStore   virtualMachineDataStore.DataStore
	virtualMachineV2Store virtualMachineV2DataStore.DataStore
}

func (p *pipelineImpl) OnFinish(_ string) {}

func (p *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return []centralsensor.CentralCapability{centralsensor.VirtualMachinesSupported}
}

func (p *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetVirtualMachine() != nil
}

func (p *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	if features.VirtualMachinesEnhancedDataModel.Enabled() {
		return p.reconcileV2(ctx, clusterID, storeMap)
	}
	return p.reconcileV1(ctx, clusterID, storeMap)
}

func (p *pipelineImpl) reconcileV1(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	query.Pagination = &v1.QueryPagination{Limit: math.MaxInt32}
	virtualMachines, err := p.virtualMachineStore.SearchRawVirtualMachines(ctx, query)
	if err != nil {
		return errors.Wrap(err, "retrieving virtual machines for reconciliation")
	}
	clusterVMIDs := set.NewStringSet()
	for _, vm := range virtualMachines {
		clusterVMIDs.Add(vm.GetId())
	}

	store := storeMap.Get((*central.SensorEvent_VirtualMachine)(nil))
	return reconciliation.Perform(store, clusterVMIDs, "virtualmachines", func(id string) error {
		return p.virtualMachineStore.DeleteVirtualMachines(ctx, id)
	})
}

func (p *pipelineImpl) reconcileV2(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	query.Pagination = &v1.QueryPagination{Limit: math.MaxInt32}
	results, err := p.virtualMachineV2Store.Search(ctx, query)
	if err != nil {
		return errors.Wrap(err, "retrieving v2 virtual machines for reconciliation")
	}

	store := storeMap.Get((*central.SensorEvent_VirtualMachine)(nil))
	return reconciliation.Perform(store, search.ResultsToIDSet(results), "virtualmachines", func(id string) error {
		return p.virtualMachineV2Store.DeleteVirtualMachines(ctx, id)
	})
}

func (p *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.VirtualMachine)

	event := msg.GetEvent()
	virtualMachine := event.GetVirtualMachine()
	if virtualMachine == nil {
		return errors.Errorf("unexpected resource type %T for virtual machine", event.GetResource())
	}

	if features.VirtualMachinesEnhancedDataModel.Enabled() {
		return p.runV2(ctx, clusterID, event.GetAction(), virtualMachine)
	}
	return p.runV1(ctx, clusterID, event.GetAction(), virtualMachine)
}

func (p *pipelineImpl) runV1(ctx context.Context, clusterID string, action central.ResourceAction, vm *virtualMachineV1.VirtualMachine) error {
	switch action {
	case central.ResourceAction_REMOVE_RESOURCE:
		return p.virtualMachineStore.DeleteVirtualMachines(ctx, vm.GetId())
	case central.ResourceAction_CREATE_RESOURCE, central.ResourceAction_UPDATE_RESOURCE, central.ResourceAction_SYNC_RESOURCE:
		return p.runUpsertPipelineV1(ctx, clusterID, vm)
	default:
		return fmt.Errorf("event action '%s' for virtual machine does not exist", action)
	}
}

func (p *pipelineImpl) runV2(ctx context.Context, clusterID string, action central.ResourceAction, vm *virtualMachineV1.VirtualMachine) error {
	switch action {
	case central.ResourceAction_REMOVE_RESOURCE:
		return p.virtualMachineV2Store.DeleteVirtualMachines(ctx, vm.GetId())
	case central.ResourceAction_CREATE_RESOURCE, central.ResourceAction_UPDATE_RESOURCE, central.ResourceAction_SYNC_RESOURCE:
		return p.runUpsertPipelineV2(ctx, clusterID, vm)
	default:
		return fmt.Errorf("event action '%s' for virtual machine does not exist", action)
	}
}

func (p *pipelineImpl) runUpsertPipelineV1(
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
	vmToStore := internaltostorage.VirtualMachineV2(vm)

	clusterName, ok, err := p.clusterStore.GetClusterName(ctx, clusterID)
	if err == nil && ok {
		vmToStore.ClusterName = clusterName
	}

	return p.virtualMachineV2Store.UpsertVirtualMachine(ctx, vmToStore)
}
