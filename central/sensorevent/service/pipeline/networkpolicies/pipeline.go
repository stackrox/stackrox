package networkpolicies

import (
	"fmt"

	clusterDataStore "bitbucket.org/stack-rox/apollo/central/cluster/datastore"
	networkPoliciesStore "bitbucket.org/stack-rox/apollo/central/networkpolicies/store"
	"bitbucket.org/stack-rox/apollo/central/sensorevent/service/pipeline"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Template design pattern. We define control flow here and defer logic to subclasses.
//////////////////////////////////////////////////////////////////////////////////////

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(clusters clusterDataStore.DataStore, networkPolicies networkPoliciesStore.Store) pipeline.Pipeline {
	return &pipelineImpl{
		clusters:        clusters,
		networkPolicies: networkPolicies,
	}
}

type pipelineImpl struct {
	clusters        clusterDataStore.DataStore
	networkPolicies networkPoliciesStore.Store
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(event *v1.SensorEvent) (*v1.SensorEventResponse, error) {
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
func (s *pipelineImpl) runRemovePipeline(action v1.ResourceAction, event *v1.NetworkPolicy) (*v1.SensorEventResponse, error) {
	// Validate the the event we receive has necessary fields set.
	if err := s.validateInput(event); err != nil {
		return nil, err
	}

	// Add/Update/Remove the deployment from persistence depending on the event action.
	if err := s.persistNetworkPolicy(action, event); err != nil {
		return nil, err
	}

	return s.createResponse(event), nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runGeneralPipeline(action v1.ResourceAction, np *v1.NetworkPolicy) (*v1.SensorEventResponse, error) {
	if err := s.validateInput(np); err != nil {
		return nil, err
	}

	if err := s.enrichCluster(np); err != nil {
		return nil, err
	}

	if err := s.persistNetworkPolicy(action, np); err != nil {
		return nil, err
	}

	return s.createResponse(np), nil
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
	case v1.ResourceAction_PREEXISTING_RESOURCE, v1.ResourceAction_CREATE_RESOURCE:
		return s.networkPolicies.AddNetworkPolicy(np)
	case v1.ResourceAction_UPDATE_RESOURCE:
		return s.networkPolicies.UpdateNetworkPolicy(np)
	case v1.ResourceAction_REMOVE_RESOURCE:
		return s.networkPolicies.RemoveNetworkPolicy(string(np.GetId()))
	default:
		return fmt.Errorf("Event action '%s' for network policy does not exist", action)
	}
}

func (s *pipelineImpl) createResponse(np *v1.NetworkPolicy) *v1.SensorEventResponse {
	return &v1.SensorEventResponse{
		Resource: &v1.SensorEventResponse_NetworkPolicy{
			NetworkPolicy: &v1.NetworkPolicyEventResponse{
				Id: string(np.GetId()),
			},
		},
	}
}
