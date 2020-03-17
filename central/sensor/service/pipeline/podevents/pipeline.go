package podevents

import (
	"context"

	countMetrics "github.com/stackrox/rox/central/metrics"
	podDataStore "github.com/stackrox/rox/central/pod/datastore"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
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
		return s.runGeneralPipeline(ctx, event.GetAction(), pod)
	}
}

func (s *pipelineImpl) runGeneralPipeline(ctx context.Context, _ central.ResourceAction, pod *storage.Pod) error {
	if err := s.pods.UpsertPod(ctx, pod); err != nil {
		return err
	}

	log.Debugf("Upserted Pod: %+v", pod)

	return nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runRemovePipeline(ctx context.Context, pod *storage.Pod) error {
	// Remove the pod from persistence.
	if err := s.pods.RemovePod(ctx, pod.GetId()); err != nil {
		return err
	}

	log.Debugf("Removed Pod: %+v", pod)

	// TODO: Add to PodHistory. Perhaps add that functionality to lifecycle manager.

	return nil
}

func (s *pipelineImpl) OnFinish(_ string) {}
