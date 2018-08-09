package namespaces

import (
	"fmt"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/store"
	"github.com/stackrox/rox/central/networkgraph"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Template design pattern. We define control flow here and defer logic to subclasses.
//////////////////////////////////////////////////////////////////////////////////////

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(clusters clusterDataStore.DataStore, namespaces namespaceDataStore.Store, graphEvaluator networkgraph.Evaluator) pipeline.Pipeline {
	return &pipelineImpl{
		clusters:       clusters,
		namespaces:     namespaces,
		graphEvaluator: graphEvaluator,
	}
}

type pipelineImpl struct {
	clusters       clusterDataStore.DataStore
	namespaces     namespaceDataStore.Store
	graphEvaluator networkgraph.Evaluator
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(event *v1.SensorEvent) (*v1.SensorEventResponse, error) {
	namespace := event.GetNamespace()
	namespace.ClusterId = event.GetClusterId()
	switch event.GetAction() {
	case v1.ResourceAction_REMOVE_RESOURCE:
		return s.runRemovePipeline(event.GetAction(), namespace)
	default:
		return s.runGeneralPipeline(event.GetAction(), namespace)
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runRemovePipeline(action v1.ResourceAction, event *v1.Namespace) (*v1.SensorEventResponse, error) {
	// Validate the the event we receive has necessary fields set.
	if err := s.validateInput(event); err != nil {
		return nil, err
	}

	// Add/Update/Remove the deployment from persistence depending on the event action.
	if err := s.persistNamespace(action, event); err != nil {
		return nil, err
	}
	s.graphEvaluator.IncrementEpoch()

	return s.createResponse(event), nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runGeneralPipeline(action v1.ResourceAction, ns *v1.Namespace) (*v1.SensorEventResponse, error) {
	if err := s.validateInput(ns); err != nil {
		return nil, err
	}

	if err := s.enrichCluster(ns); err != nil {
		return nil, err
	}

	if err := s.persistNamespace(action, ns); err != nil {
		return nil, err
	}
	s.graphEvaluator.IncrementEpoch()

	return s.createResponse(ns), nil
}

func (s *pipelineImpl) validateInput(np *v1.Namespace) error {
	// validate input.
	if np == nil {
		return fmt.Errorf("namespace must not be empty")
	}
	return nil
}

func (s *pipelineImpl) enrichCluster(ns *v1.Namespace) error {
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

func (s *pipelineImpl) persistNamespace(action v1.ResourceAction, ns *v1.Namespace) error {
	switch action {
	case v1.ResourceAction_PREEXISTING_RESOURCE, v1.ResourceAction_CREATE_RESOURCE:
		return s.namespaces.AddNamespace(ns)
	case v1.ResourceAction_UPDATE_RESOURCE:
		return s.namespaces.UpdateNamespace(ns)
	case v1.ResourceAction_REMOVE_RESOURCE:
		return s.namespaces.RemoveNamespace(ns.GetId())
	default:
		return fmt.Errorf("Event action '%s' for namespace does not exist", action)
	}
}

func (s *pipelineImpl) createResponse(ns *v1.Namespace) *v1.SensorEventResponse {
	return &v1.SensorEventResponse{
		Resource: &v1.SensorEventResponse_Namespace{
			Namespace: &v1.NamespaceEventResponse{
				Id: ns.Id,
			},
		},
	}
}
