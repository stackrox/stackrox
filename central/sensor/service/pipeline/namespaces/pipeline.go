package namespaces

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	countMetrics "github.com/stackrox/rox/central/metrics"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	"github.com/stackrox/rox/central/networkpolicies/graph"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()

	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

// Template design pattern. We define control flow here and defer logic to subclasses.
//////////////////////////////////////////////////////////////////////////////////////

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(clusterDataStore.Singleton(), namespaceDataStore.Singleton(), graph.Singleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(clusters clusterDataStore.DataStore, namespaces namespaceDataStore.DataStore, graphEvaluator graph.Evaluator) pipeline.Fragment {
	return &pipelineImpl{
		clusters:       clusters,
		namespaces:     namespaces,
		graphEvaluator: graphEvaluator,
	}
}

type pipelineImpl struct {
	clusters       clusterDataStore.DataStore
	namespaces     namespaceDataStore.DataStore
	graphEvaluator graph.Evaluator
}

func (s *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	results, err := s.namespaces.Search(ctx, query)
	if err != nil {
		return err
	}
	store := storeMap.Get((*central.SensorEvent_Namespace)(nil))
	return reconciliation.Perform(store, search.ResultsToIDSet(results), "namespaces", func(id string) error {
		return s.runRemovePipeline(ctx, central.ResourceAction_REMOVE_RESOURCE, &storage.NamespaceMetadata{Id: id})
	})
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetNamespace() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.Namespace)

	event := msg.GetEvent()
	namespace := event.GetNamespace()
	namespace.ClusterId = clusterID

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.runRemovePipeline(ctx, event.GetAction(), namespace)
	default:
		return s.runGeneralPipeline(ctx, event.GetAction(), namespace)
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runRemovePipeline(ctx context.Context, action central.ResourceAction, event *storage.NamespaceMetadata) error {
	// Validate the the event we receive has necessary fields set.
	if err := s.validateInput(event); err != nil {
		return err
	}

	// Add/Update/Remove the namespace from persistence depending on the event action.
	if err := s.persistNamespace(ctx, action, event); err != nil {
		return err
	}
	s.graphEvaluator.IncrementEpoch(event.GetClusterId())

	return nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runGeneralPipeline(ctx context.Context, action central.ResourceAction, ns *storage.NamespaceMetadata) error {
	if err := s.validateInput(ns); err != nil {
		return err
	}

	if err := s.enrichCluster(ctx, ns); err != nil {
		return err
	}

	if err := s.persistNamespace(ctx, action, ns); err != nil {
		return err
	}
	s.graphEvaluator.IncrementEpoch(ns.GetClusterId())

	return nil
}

func (s *pipelineImpl) validateInput(np *storage.NamespaceMetadata) error {
	// validate input.
	if np == nil {
		return errors.New("namespace must not be empty")
	}
	return nil
}

func (s *pipelineImpl) enrichCluster(ctx context.Context, ns *storage.NamespaceMetadata) error {
	ns.ClusterName = ""

	clusterName, clusterExists, err := s.clusters.GetClusterName(ctx, ns.ClusterId)
	switch {
	case err != nil:
		log.Warnf("Couldn't get name of cluster: %s", err)
	case !clusterExists:
		log.Warnf("Couldn't find cluster '%s'", ns.ClusterId)
	default:
		ns.ClusterName = clusterName
	}
	return nil
}

func (s *pipelineImpl) persistNamespace(ctx context.Context, action central.ResourceAction, ns *storage.NamespaceMetadata) error {
	switch action {
	// Sync resource is treated as a CREATE because it updates the idMap which is used for SAC within the AddNamespace Function
	case central.ResourceAction_CREATE_RESOURCE, central.ResourceAction_SYNC_RESOURCE:
		return s.namespaces.AddNamespace(ctx, ns)
	case central.ResourceAction_UPDATE_RESOURCE:
		return s.namespaces.UpdateNamespace(ctx, ns)
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.namespaces.RemoveNamespace(ctx, ns.GetId())
	default:
		return fmt.Errorf("Event action '%s' for namespace does not exist", action)
	}
}

func (s *pipelineImpl) OnFinish(_ string) {}
