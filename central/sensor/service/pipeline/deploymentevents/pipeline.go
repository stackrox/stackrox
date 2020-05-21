package deploymentevents

import (
	"context"

	"github.com/mitchellh/hashstructure"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/lifecycle"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/networkpolicies/graph"
	"github.com/stackrox/rox/central/reprocessor"
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
	return NewPipeline(clusterDataStore.Singleton(),
		deploymentDataStore.Singleton(),
		imageDataStore.Singleton(),
		lifecycle.SingletonManager(),
		graph.Singleton(),
		reprocessor.Singleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(clusters clusterDataStore.DataStore, deployments deploymentDataStore.DataStore,
	images imageDataStore.DataStore, manager lifecycle.Manager,
	graphEvaluator graph.Evaluator, reprocessor reprocessor.Loop) pipeline.Fragment {
	return &pipelineImpl{
		validateInput:     newValidateInput(),
		clusterEnrichment: newClusterEnrichment(clusters),
		updateImages:      newUpdateImages(images),
		lifecycleManager:  manager,

		graphEvaluator: graphEvaluator,
		deployments:    deployments,
		clusters:       clusters,

		reprocessor: reprocessor,
	}
}

type pipelineImpl struct {
	// pipeline stages.
	validateInput     *validateInputImpl
	clusterEnrichment *clusterEnrichmentImpl
	updateImages      *updateImagesImpl
	lifecycleManager  lifecycle.Manager

	deployments deploymentDataStore.DataStore
	clusters    clusterDataStore.DataStore
	reprocessor reprocessor.Loop

	graphEvaluator graph.Evaluator
}

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	results, err := s.deployments.Search(ctx, query)
	if err != nil {
		return err
	}

	store := storeMap.Get((*central.SensorEvent_Deployment)(nil))
	return reconciliation.Perform(store, search.ResultsToIDSet(results), "deployments", func(id string) error {
		return s.runRemovePipeline(ctx, &storage.Deployment{Id: id}, true)
	})
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetDeployment() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.Deployment)

	event := msg.GetEvent()
	deployment := event.GetDeployment()
	deployment.ClusterId = clusterID

	var err error
	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		err = s.runRemovePipeline(ctx, deployment, false)
	default:
		err = s.runGeneralPipeline(ctx, deployment)
	}
	return err
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runRemovePipeline(ctx context.Context, deployment *storage.Deployment, reconciliation bool) error {
	// Validate the the deployment we receive has necessary fields set.
	if err := s.validateInput.do(deployment); err != nil {
		return err
	}

	// Remove the deployment from persistence.
	if err := s.deployments.RemoveDeployment(ctx, deployment.GetClusterId(), deployment.GetId()); err != nil {
		return err
	}

	s.graphEvaluator.IncrementEpoch(deployment.GetClusterId())
	if reconciliation {
		// Only remove the alerts during reconciliation as the sensor will naturally send an empty AlertResults
		// struct which will properly clean up the existing deploy time alerts
		if err := s.lifecycleManager.DeploymentRemoved(deployment); err != nil {
			return err
		}
	}
	return nil
}

func compareMap(m1, m2 map[string]string) bool {
	if len(m1) != len(m2) {
		return false
	}
	for k, v1 := range m1 {
		if v2, ok := m2[k]; !ok || v2 != v1 {
			return false
		}
	}
	return true
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runGeneralPipeline(ctx context.Context, deployment *storage.Deployment) error {
	// Validate the the deployment we receive has necessary fields set.
	if err := s.validateInput.do(deployment); err != nil {
		return err
	}

	var err error
	deployment.Hash, err = hashstructure.Hash(deployment, &hashstructure.HashOptions{})
	if err != nil {
		return err
	}

	// Fill in cluster information.
	if err := s.clusterEnrichment.do(ctx, deployment); err != nil {
		log.Errorf("Couldn't get cluster identity: %s", err)
		return err
	}

	oldDeployment, exists, err := s.deployments.GetDeployment(ctx, deployment.GetId())
	if err != nil {
		return err
	}
	incrementNetworkGraphEpoch := true

	// If it exists, check to see if we can dedupe it
	if exists {
		if oldDeployment.GetHash() == deployment.GetHash() {
			// There is a separate handler for ContainerInstances,
			// so there is no longer a need to continue from this point.
			// This will only be reached upon a re-sync event from k8s.
			return nil
		}
		incrementNetworkGraphEpoch = !compareMap(oldDeployment.GetPodLabels(), deployment.GetPodLabels())
	}

	// Add/Update the deployment from persistence depending on the deployment action.
	if err := s.deployments.UpsertDeployment(ctx, deployment); err != nil {
		return err
	}

	// Update risk asynchronously
	s.reprocessor.ReprocessRiskForDeployments(deployment.GetId())

	if incrementNetworkGraphEpoch {
		s.graphEvaluator.IncrementEpoch(deployment.GetClusterId())
	}
	return nil
}

func (s *pipelineImpl) OnFinish(_ string) {}
