package deploymentevents

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/mitchellh/hashstructure"
	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/lifecycle"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/networkpolicies/graph"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/enforcers"
	"github.com/stackrox/rox/pkg/features"
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
		graph.Singleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(clusters clusterDataStore.DataStore, deployments deploymentDataStore.DataStore,
	images imageDataStore.DataStore, manager lifecycle.Manager,
	graphEvaluator graph.Evaluator) pipeline.Fragment {
	return &pipelineImpl{
		validateInput:     newValidateInput(),
		clusterEnrichment: newClusterEnrichment(clusters),
		updateImages:      newUpdateImages(images),
		persistDeployment: newPersistDeployment(deployments),
		createResponse:    newCreateResponse(manager.DeploymentUpdated, manager.DeploymentRemoved),

		graphEvaluator: graphEvaluator,
		deployments:    deployments,
		clusters:       clusters,
		reconcileStore: reconciliation.NewStore(),
	}
}

type pipelineImpl struct {
	// pipeline stages.
	validateInput     *validateInputImpl
	clusterEnrichment *clusterEnrichmentImpl
	updateImages      *updateImagesImpl
	persistDeployment *persistDeploymentImpl
	createResponse    *createResponseImpl

	deployments deploymentDataStore.DataStore
	clusters    clusterDataStore.DataStore

	graphEvaluator graph.Evaluator

	reconcileStore reconciliation.Store
}

func (s *pipelineImpl) Reconcile(clusterID string) error {
	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	results, err := s.deployments.Search(context.TODO(), query)
	if err != nil {
		return err
	}

	return reconciliation.PerformDryRun(s.reconcileStore, search.ResultsToIDSet(results), "deployments", func(id string) error {
		_, err := s.runRemovePipeline(central.ResourceAction_REMOVE_RESOURCE, &storage.Deployment{Id: id})
		return err
	}, !features.PerformDeploymentReconciliation.Enabled())
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetDeployment() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(clusterID string, msg *central.MsgFromSensor, injector common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.Deployment)

	event := msg.GetEvent()
	deployment := event.GetDeployment()
	deployment.ClusterId = clusterID

	var resp *central.SensorEnforcement
	var err error
	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		resp, err = s.runRemovePipeline(event.GetAction(), deployment)
	default:
		s.reconcileStore.Add(event.GetId())
		resp, err = s.runGeneralPipeline(event.GetAction(), deployment)
	}
	if err != nil {
		return err
	}
	if resp != nil {
		if enforcers.ShouldEnforce(deployment.GetAnnotations()) {
			err := injector.InjectMessage(context.Background(), &central.MsgToSensor{
				Msg: &central.MsgToSensor_Enforcement{
					Enforcement: resp,
				},
			})
			if err != nil {
				log.Errorf("Failed to inject enforcement action %s: %v", proto.MarshalTextString(resp), err)
			}
		} else {
			log.Warnf("Did not inject enforcement because deployment %s contained Enforcement Bypass annotations", deployment.GetName())
		}
	}
	return nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runRemovePipeline(action central.ResourceAction, deployment *storage.Deployment) (*central.SensorEnforcement, error) {
	// Validate the the deployment we receive has necessary fields set.
	if err := s.validateInput.do(deployment); err != nil {
		return nil, err
	}

	// Add/Update/Remove the deployment from persistence depending on the deployment action.
	if err := s.persistDeployment.do(action, deployment); err != nil {
		return nil, err
	}

	s.graphEvaluator.IncrementEpoch()

	// Process the deployment (enrichment, alert generation, enforcement action generation.)
	resp := s.createResponse.do(deployment, action)
	return resp, nil
}

func computeDeploymentHashWithoutContainerInstances(d *storage.Deployment) error {
	d.Hash = 0
	containerInstances := make([][]*storage.ContainerInstance, 0, len(d.GetContainers()))
	for _, c := range d.GetContainers() {
		containerInstances = append(containerInstances, c.GetInstances())
		c.Instances = nil
	}
	var err error
	d.Hash, err = hashstructure.Hash(d, &hashstructure.HashOptions{})

	for i, c := range d.GetContainers() {
		c.Instances = containerInstances[i]
	}
	return err
}

func (s *pipelineImpl) dedupeBasedOnHash(action central.ResourceAction, newDeployment *storage.Deployment) (bool, error) {
	if err := computeDeploymentHashWithoutContainerInstances(newDeployment); err != nil {
		return false, err
	}

	// Check if this deployment needs to be processed based on hash
	oldDeployment, exists, err := s.deployments.GetDeployment(context.TODO(), newDeployment.GetId())
	if err != nil {
		return false, err
	}
	// If it already exists and the hash is the same, then just update the container instances of the old deployment and upsert
	if exists && oldDeployment.GetHash() == newDeployment.GetHash() {
		// Using the index of Container is save as this is ensured by the hash
		for i, c := range newDeployment.GetContainers() {
			oldDeployment.Containers[i].Instances = c.Instances
		}
		return true, s.persistDeployment.do(action, oldDeployment)
	}
	return false, nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runGeneralPipeline(action central.ResourceAction, deployment *storage.Deployment) (*central.SensorEnforcement, error) {
	// Validate the the deployment we receive has necessary fields set.
	if err := s.validateInput.do(deployment); err != nil {
		return nil, err
	}

	dedupe, err := s.dedupeBasedOnHash(action, deployment)
	if err != nil {
		err = errors.Wrapf(err, "Could not check deployment %q for deduping", deployment.GetName())
		log.Error(err)
		return nil, err
	}
	if dedupe {
		return nil, nil
	}

	// Fill in cluster information.
	if err := s.clusterEnrichment.do(deployment); err != nil {
		log.Errorf("Couldn't get cluster identity: %s", err)
	}

	// Add/Update/Remove the deployment from persistence depending on the deployment action.
	if err := s.persistDeployment.do(action, deployment); err != nil {
		return nil, err
	}

	// Update the deployments images with the latest version from storage.
	s.updateImages.do(deployment)

	// Process the deployment (alert generation, enforcement action generation)
	resp := s.createResponse.do(deployment, action)

	// We want to persist the images from the deployment in the deployment after processing (create response)
	// TODO(rs): We should map out how images are updated in the pipeline so we don't do more writes than needed.
	s.updateImages.do(deployment)

	s.graphEvaluator.IncrementEpoch()

	return resp, nil
}

func (s *pipelineImpl) OnFinish(clusterID string) {}
