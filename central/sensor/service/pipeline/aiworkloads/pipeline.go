package aiworkloads

import (
	"context"
	"fmt"
	"math"

	"github.com/pkg/errors"
	aiWorkloadDataStore "github.com/stackrox/rox/central/aiworkload/datastore"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/convert/internaltostorage"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

var _ pipeline.Fragment = (*pipelineImpl)(nil)

func GetPipeline() pipeline.Fragment {
	return newPipeline(clusterDataStore.Singleton(), aiWorkloadDataStore.Singleton())
}

func newPipeline(clusterStore clusterDataStore.DataStore, aiWorkloadStore aiWorkloadDataStore.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		clusterStore:    clusterStore,
		aiWorkloadStore: aiWorkloadStore,
	}
}

type pipelineImpl struct {
	clusterStore    clusterDataStore.DataStore
	aiWorkloadStore aiWorkloadDataStore.DataStore
}

func (p *pipelineImpl) OnFinish(_ string) {}

func (p *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return []centralsensor.CentralCapability{centralsensor.AIWorkloadsSupported}
}

func (p *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetAiWorkload() != nil
}

func (p *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	query.Pagination = &v1.QueryPagination{Limit: math.MaxInt32}
	workloads, err := p.aiWorkloadStore.SearchRawAIWorkloads(ctx, query)
	if err != nil {
		return errors.Wrap(err, "retrieving AI workloads for reconciliation")
	}
	clusterWorkloadIDs := set.NewStringSet()
	for _, w := range workloads {
		clusterWorkloadIDs.Add(w.GetId())
	}

	store := storeMap.Get((*central.SensorEvent_AiWorkload)(nil))
	return reconciliation.Perform(store, clusterWorkloadIDs, "aiworkloads", func(id string) error {
		return p.aiWorkloadStore.DeleteAIWorkloads(ctx, id)
	})
}

func (p *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.AIWorkload)

	event := msg.GetEvent()
	aiWorkload := event.GetAiWorkload()
	if aiWorkload == nil {
		return errors.Errorf("unexpected resource type %T for AI workload", event.GetResource())
	}

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return p.aiWorkloadStore.DeleteAIWorkloads(ctx, aiWorkload.GetId())
	case central.ResourceAction_CREATE_RESOURCE, central.ResourceAction_UPDATE_RESOURCE, central.ResourceAction_SYNC_RESOURCE:
		workloadToStore := internaltostorage.AIWorkload(aiWorkload)
		clusterName, ok, err := p.clusterStore.GetClusterName(ctx, clusterID)
		if err != nil {
			return errors.Wrap(err, "retrieving cluster name for AI workload")
		}
		if ok {
			workloadToStore.ClusterName = clusterName
		}
		return p.aiWorkloadStore.UpsertAIWorkload(ctx, workloadToStore)
	default:
		return fmt.Errorf("event action '%s' for AI workload does not exist", event.GetAction())
	}
}
