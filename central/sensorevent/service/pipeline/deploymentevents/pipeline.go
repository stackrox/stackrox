package deploymentevents

import (
	"github.com/gogo/protobuf/proto"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/lifecycle"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/networkpolicies/graph"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Template design pattern. We define control flow here and defer logic to subclasses.
//////////////////////////////////////////////////////////////////////////////////////

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(clusters clusterDataStore.DataStore, deployments deploymentDataStore.DataStore,
	images imageDataStore.DataStore, manager lifecycle.Manager,
	graphEvaluator graph.Evaluator) pipeline.Pipeline {
	return &pipelineImpl{
		validateInput:     newValidateInput(),
		clusterEnrichment: newClusterEnrichment(clusters),
		updateImages:      newUpdateImages(images),
		persistDeployment: newPersistDeployment(deployments),
		createResponse:    newCreateResponse(manager.DeploymentUpdated, manager.DeploymentRemoved),

		graphEvaluator: graphEvaluator,
	}
}

type pipelineImpl struct {
	// pipeline stages.
	validateInput     *validateInputImpl
	clusterEnrichment *clusterEnrichmentImpl
	updateImages      *updateImagesImpl
	persistDeployment *persistDeploymentImpl
	createResponse    *createResponseImpl

	graphEvaluator graph.Evaluator
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(event *v1.SensorEvent, injector pipeline.EnforcementInjector) error {
	deployment := event.GetDeployment()
	deployment.ClusterId = event.GetClusterId()

	var resp *v1.SensorEnforcement
	var err error
	switch event.GetAction() {
	case v1.ResourceAction_REMOVE_RESOURCE:
		resp, err = s.runRemovePipeline(event.GetAction(), deployment)
	default:
		resp, err = s.runGeneralPipeline(event.GetAction(), deployment)
	}
	if err != nil {
		return err
	}
	if resp != nil {
		injected := injector.InjectEnforcement(resp)
		if !injected {
			log.Errorf("Failed to inject enforcement action %s", proto.MarshalTextString(resp))
		}
	}
	return nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runRemovePipeline(action v1.ResourceAction, deployment *storage.Deployment) (*v1.SensorEnforcement, error) {
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

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runGeneralPipeline(action v1.ResourceAction, deployment *storage.Deployment) (*v1.SensorEnforcement, error) {
	// Validate the the deployment we receive has necessary fields set.
	if err := s.validateInput.do(deployment); err != nil {
		return nil, err
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
