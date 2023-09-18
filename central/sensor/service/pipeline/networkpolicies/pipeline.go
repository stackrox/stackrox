package networkpolicies

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	countMetrics "github.com/stackrox/rox/central/metrics"
	npDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	"github.com/stackrox/rox/central/networkpolicies/graph"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/set"
)

var (
	log = logging.LoggerForModule()

	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

// Template design pattern. We define control flow here and defer logic to subclasses.
//////////////////////////////////////////////////////////////////////////////////////

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(clusterDataStore.Singleton(), npDS.Singleton(), graph.Singleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(clusters clusterDataStore.DataStore, networkPolicies npDS.DataStore, graphEvaluator graph.Evaluator) pipeline.Fragment {
	return &pipelineImpl{
		clusters:        clusters,
		networkPolicies: networkPolicies,
		graphEvaluator:  graphEvaluator,
	}
}

type pipelineImpl struct {
	clusters        clusterDataStore.DataStore
	networkPolicies npDS.DataStore
	graphEvaluator  graph.Evaluator
}

func (s *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	networkPolicies, err := s.networkPolicies.GetNetworkPolicies(ctx, clusterID, "")
	if err != nil {
		return err
	}

	existingIDs := set.NewStringSet()
	for _, n := range networkPolicies {
		existingIDs.Add(n.GetId())
	}
	store := storeMap.Get((*central.SensorEvent_NetworkPolicy)(nil))
	return reconciliation.Perform(store, existingIDs, "network policies", func(id string) error {
		return s.runRemovePipeline(ctx, central.ResourceAction_REMOVE_RESOURCE, &storage.NetworkPolicy{Id: id})
	})
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetNetworkPolicy() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.NetworkPolicy)

	event := msg.GetEvent()
	networkPolicy := event.GetNetworkPolicy()
	networkPolicy.ClusterId = clusterID

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.runRemovePipeline(ctx, event.GetAction(), networkPolicy)
	default:
		return s.runGeneralPipeline(ctx, event.GetAction(), networkPolicy)
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runRemovePipeline(ctx context.Context, action central.ResourceAction, event *storage.NetworkPolicy) error {
	// Validate the the event we receive has necessary fields set.
	if err := s.validateInput(event); err != nil {
		return err
	}

	// Add/Update/Remove the network policy from persistence depending on the event action.
	if err := s.persistNetworkPolicy(ctx, action, event); err != nil {
		return err
	}
	s.graphEvaluator.IncrementEpoch(event.GetClusterId())

	return nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runGeneralPipeline(ctx context.Context, action central.ResourceAction, np *storage.NetworkPolicy) error {
	if err := s.validateInput(np); err != nil {
		return err
	}

	if err := s.enrichCluster(ctx, np); err != nil {
		return err
	}

	if err := s.persistNetworkPolicy(ctx, action, np); err != nil {
		return err
	}
	s.graphEvaluator.IncrementEpoch(np.GetClusterId())

	return nil
}

func (s *pipelineImpl) validateInput(np *storage.NetworkPolicy) error {
	// validate input.
	if np == nil {
		return errors.New("network policy must not be empty")
	}
	return nil
}

func (s *pipelineImpl) enrichCluster(ctx context.Context, np *storage.NetworkPolicy) error {
	np.ClusterName = ""

	clusterName, clusterExists, err := s.clusters.GetClusterName(ctx, np.ClusterId)
	switch {
	case err != nil:
		log.Warnf("Couldn't get name of cluster: %s", err)
	case !clusterExists:
		log.Warnf("Couldn't find cluster '%s'", np.ClusterId)
	default:
		np.ClusterName = clusterName
	}
	return nil
}

func (s *pipelineImpl) persistNetworkPolicy(ctx context.Context, action central.ResourceAction, np *storage.NetworkPolicy) error {
	switch action {
	case central.ResourceAction_CREATE_RESOURCE, central.ResourceAction_UPDATE_RESOURCE, central.ResourceAction_SYNC_RESOURCE:
		return s.networkPolicies.UpsertNetworkPolicy(ctx, np)
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.networkPolicies.RemoveNetworkPolicy(ctx, string(np.GetId()))
	default:
		return fmt.Errorf("Event action '%s' for network policy does not exist", action)
	}
}

func (s *pipelineImpl) OnFinish(_ string) {}
