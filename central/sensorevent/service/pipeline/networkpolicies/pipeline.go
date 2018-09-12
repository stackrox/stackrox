package networkpolicies

import (
	"fmt"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/networkgraph"
	networkPoliciesStore "github.com/stackrox/rox/central/networkpolicies/store"
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
func NewPipeline(clusters clusterDataStore.DataStore, networkPolicies networkPoliciesStore.Store, graphEvaluator networkgraph.Evaluator) pipeline.Pipeline {
	return &pipelineImpl{
		clusters:        clusters,
		networkPolicies: networkPolicies,
		graphEvaluator:  graphEvaluator,
	}
}

type pipelineImpl struct {
	clusters        clusterDataStore.DataStore
	networkPolicies networkPoliciesStore.Store
	graphEvaluator  networkgraph.Evaluator
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(event *v1.SensorEvent) (*v1.SensorEnforcement, error) {
	networkPolicy := event.GetNetworkPolicy()
	networkPolicy.ClusterId = event.GetClusterId()

	switch event.GetAction() {
	case v1.ResourceAction_REMOVE_RESOURCE:
		return s.runRemovePipeline(event.GetAction(), networkPolicy)
	default:
		return s.runGeneralPipeline(event.GetAction(), networkPolicy)
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runRemovePipeline(action v1.ResourceAction, event *v1.NetworkPolicy) (*v1.SensorEnforcement, error) {
	// Validate the the event we receive has necessary fields set.
	if err := s.validateInput(event); err != nil {
		return nil, err
	}

	// Add/Update/Remove the deployment from persistence depending on the event action.
	if err := s.persistNetworkPolicy(action, event); err != nil {
		return nil, err
	}
	s.graphEvaluator.IncrementEpoch()

	return nil, nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runGeneralPipeline(action v1.ResourceAction, np *v1.NetworkPolicy) (*v1.SensorEnforcement, error) {
	if err := s.validateInput(np); err != nil {
		return nil, err
	}

	if err := s.enrichCluster(np); err != nil {
		return nil, err
	}

	if err := s.persistNetworkPolicy(action, np); err != nil {
		return nil, err
	}
	s.graphEvaluator.IncrementEpoch()

	return nil, nil
}

func (s *pipelineImpl) validateInput(np *v1.NetworkPolicy) error {
	// validate input.
	if np == nil {
		return fmt.Errorf("network policy must not be empty")
	}
	return nil
}

func (s *pipelineImpl) enrichCluster(np *v1.NetworkPolicy) error {
	np.ClusterName = ""

	cluster, clusterExists, err := s.clusters.GetCluster(np.ClusterId)
	switch {
	case err != nil:
		log.Warnf("Couldn't get name of cluster: %s", err)
	case !clusterExists:
		log.Warnf("Couldn't find cluster '%s'", np.ClusterId)
	default:
		np.ClusterName = cluster.GetName()
	}
	return nil
}

func (s *pipelineImpl) persistNetworkPolicy(action v1.ResourceAction, np *v1.NetworkPolicy) error {
	switch action {
	case v1.ResourceAction_CREATE_RESOURCE:
		return s.networkPolicies.AddNetworkPolicy(np)
	case v1.ResourceAction_UPDATE_RESOURCE:
		return s.networkPolicies.UpdateNetworkPolicy(np)
	case v1.ResourceAction_REMOVE_RESOURCE:
		return s.networkPolicies.RemoveNetworkPolicy(string(np.GetId()))
	default:
		return fmt.Errorf("Event action '%s' for network policy does not exist", action)
	}
}
