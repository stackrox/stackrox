package networkpolicies

import (
	"fmt"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/networkpolicies/graph"
	networkPoliciesStore "github.com/stackrox/rox/central/networkpolicies/store"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	log = logging.LoggerForModule()
)

// Template design pattern. We define control flow here and defer logic to subclasses.
//////////////////////////////////////////////////////////////////////////////////////

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(clusters clusterDataStore.DataStore, networkPolicies networkPoliciesStore.Store, graphEvaluator graph.Evaluator) pipeline.Fragment {
	return &pipelineImpl{
		clusters:        clusters,
		networkPolicies: networkPolicies,
		graphEvaluator:  graphEvaluator,
	}
}

type pipelineImpl struct {
	clusters        clusterDataStore.DataStore
	networkPolicies networkPoliciesStore.Store
	graphEvaluator  graph.Evaluator
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetNetworkPolicy() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(msg *central.MsgFromSensor, _ pipeline.MsgInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.NetworkPolicy)

	event := msg.GetEvent()
	networkPolicy := event.GetNetworkPolicy()
	networkPolicy.ClusterId = event.GetClusterId()

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.runRemovePipeline(event.GetAction(), networkPolicy)
	default:
		return s.runGeneralPipeline(event.GetAction(), networkPolicy)
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runRemovePipeline(action central.ResourceAction, event *storage.NetworkPolicy) error {
	// Validate the the event we receive has necessary fields set.
	if err := s.validateInput(event); err != nil {
		return err
	}

	// Add/Update/Remove the deployment from persistence depending on the event action.
	if err := s.persistNetworkPolicy(action, event); err != nil {
		return err
	}
	s.graphEvaluator.IncrementEpoch()

	return nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runGeneralPipeline(action central.ResourceAction, np *storage.NetworkPolicy) error {
	if err := s.validateInput(np); err != nil {
		return err
	}

	if err := s.enrichCluster(np); err != nil {
		return err
	}

	if err := s.persistNetworkPolicy(action, np); err != nil {
		return err
	}
	s.graphEvaluator.IncrementEpoch()

	return nil
}

func (s *pipelineImpl) validateInput(np *storage.NetworkPolicy) error {
	// validate input.
	if np == nil {
		return fmt.Errorf("network policy must not be empty")
	}
	return nil
}

func (s *pipelineImpl) enrichCluster(np *storage.NetworkPolicy) error {
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

func (s *pipelineImpl) persistNetworkPolicy(action central.ResourceAction, np *storage.NetworkPolicy) error {
	switch action {
	case central.ResourceAction_CREATE_RESOURCE:
		return s.networkPolicies.AddNetworkPolicy(np)
	case central.ResourceAction_UPDATE_RESOURCE:
		return s.networkPolicies.UpdateNetworkPolicy(np)
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.networkPolicies.RemoveNetworkPolicy(string(np.GetId()))
	default:
		return fmt.Errorf("Event action '%s' for network policy does not exist", action)
	}
}

func (s *pipelineImpl) OnFinish() {}
