package podevents

import (
	"context"

	countMetrics "github.com/stackrox/stackrox/central/metrics"
	podDataStore "github.com/stackrox/stackrox/central/pod/datastore"
	"github.com/stackrox/stackrox/central/sensor/service/common"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/metrics"
	"github.com/stackrox/stackrox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(podDataStore.Singleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(store podDataStore.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		pods: store,
	}
}

type pipelineImpl struct {
	pods podDataStore.DataStore
}

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	results, err := s.pods.Search(ctx, query)
	if err != nil {
		return err
	}

	log.Debugf("Reconcile search results: %+v", results)

	store := storeMap.Get((*central.SensorEvent_Pod)(nil))
	return reconciliation.Perform(store, search.ResultsToIDSet(results), "pods", func(id string) error {
		return s.runRemovePipeline(ctx, &storage.Pod{Id: id})
	})
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetPod() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, _ string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.Pod)

	event := msg.GetEvent()
	pod := event.GetPod()

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.runRemovePipeline(ctx, pod)
	default:
		// Create and Update events solely trigger an upsert.
		return s.pods.UpsertPod(ctx, pod)
	}
}

func (s *pipelineImpl) runRemovePipeline(ctx context.Context, pod *storage.Pod) error {
	// Remove the pod from persistence.
	return s.pods.RemovePod(ctx, pod.GetId())
}

func (s *pipelineImpl) OnFinish(_ string) {}
