package namespaces

import (
	"fmt"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	countMetrics "github.com/stackrox/rox/central/metrics"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	"github.com/stackrox/rox/central/networkpolicies/graph"
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
		reconcileStore: reconciliation.NewStore(),
	}
}

type pipelineImpl struct {
	clusters       clusterDataStore.DataStore
	namespaces     namespaceDataStore.DataStore
	graphEvaluator graph.Evaluator
	reconcileStore reconciliation.Store
}

func (s *pipelineImpl) Reconcile(clusterID string) error {
	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	results, err := s.namespaces.Search(query)
	if err != nil {
		return err
	}
	return reconciliation.Perform(s.reconcileStore, search.ResultsToIDSet(results), "namespaces", func(id string) error {
		return s.runRemovePipeline(central.ResourceAction_REMOVE_RESOURCE, &storage.NamespaceMetadata{Id: id})
	})
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetNamespace() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.Namespace)

	event := msg.GetEvent()
	namespace := event.GetNamespace()
	namespace.ClusterId = clusterID

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.runRemovePipeline(event.GetAction(), namespace)
	default:
		s.reconcileStore.Add(event.GetId())
		return s.runGeneralPipeline(event.GetAction(), namespace)
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runRemovePipeline(action central.ResourceAction, event *storage.NamespaceMetadata) error {
	// Validate the the event we receive has necessary fields set.
	if err := s.validateInput(event); err != nil {
		return err
	}

	// Add/Update/Remove the deployment from persistence depending on the event action.
	if err := s.persistNamespace(action, event); err != nil {
		return err
	}
	s.graphEvaluator.IncrementEpoch()

	return nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runGeneralPipeline(action central.ResourceAction, ns *storage.NamespaceMetadata) error {
	if err := s.validateInput(ns); err != nil {
		return err
	}

	if err := s.enrichCluster(ns); err != nil {
		return err
	}

	if err := s.persistNamespace(action, ns); err != nil {
		return err
	}
	s.graphEvaluator.IncrementEpoch()

	return nil
}

func (s *pipelineImpl) validateInput(np *storage.NamespaceMetadata) error {
	// validate input.
	if np == nil {
		return fmt.Errorf("namespace must not be empty")
	}
	return nil
}

func (s *pipelineImpl) enrichCluster(ns *storage.NamespaceMetadata) error {
	ns.ClusterName = ""

	cluster, clusterExists, err := s.clusters.GetCluster(ns.ClusterId)
	switch {
	case err != nil:
		log.Warnf("Couldn't get name of cluster: %s", err)
	case !clusterExists:
		log.Warnf("Couldn't find cluster '%s'", ns.ClusterId)
	default:
		ns.ClusterName = cluster.GetName()
	}
	return nil
}

func (s *pipelineImpl) persistNamespace(action central.ResourceAction, ns *storage.NamespaceMetadata) error {
	switch action {
	case central.ResourceAction_CREATE_RESOURCE:
		return s.namespaces.AddNamespace(ns)
	case central.ResourceAction_UPDATE_RESOURCE:
		return s.namespaces.UpdateNamespace(ns)
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.namespaces.RemoveNamespace(ns.GetId())
	default:
		return fmt.Errorf("Event action '%s' for namespace does not exist", action)
	}
}

func (s *pipelineImpl) OnFinish() {}
