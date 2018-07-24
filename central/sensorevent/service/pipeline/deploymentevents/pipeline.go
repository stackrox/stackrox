package deploymentevents

import (
	clusterDataStore "bitbucket.org/stack-rox/apollo/central/cluster/datastore"
	deploymentDataStore "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	"bitbucket.org/stack-rox/apollo/central/detection"
	imageDataStore "bitbucket.org/stack-rox/apollo/central/image/datastore"
	secretDataGraph "bitbucket.org/stack-rox/apollo/central/secret/datagraph"
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
func NewPipeline(clusters clusterDataStore.DataStore, deployments deploymentDataStore.DataStore, images imageDataStore.DataStore, detector detection.Detector) pipeline.Pipeline {
	return &pipelineImpl{
		validateInput:     newValidateInput(),
		clusterEnrichment: newClusterEnrichment(clusters),
		updateImages:      newUpdateImages(images),
		persistDeployment: newPersistDeployment(deployments),
		createResponse:    newCreateResponse(detector.ProcessDeploymentEvent),
	}
}

type pipelineImpl struct {
	// pipeline stages.
	validateInput     *validateInputImpl
	clusterEnrichment *clusterEnrichmentImpl
	updateImages      *updateImagesImpl
	persistDeployment *persistDeploymentImpl
	createResponse    *createResponseImpl
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(event *v1.SensorEvent) (*v1.SensorEventResponse, error) {
	deployment := event.GetDeployment()
	deployment.ClusterId = event.GetClusterId()
	switch event.GetAction() {
	case v1.ResourceAction_REMOVE_RESOURCE:
		return s.runRemovePipeline(event.GetAction(), deployment)
	default:
		return s.runGeneralPipeline(event.GetAction(), deployment)
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runRemovePipeline(action v1.ResourceAction, deployment *v1.Deployment) (*v1.SensorEventResponse, error) {
	// Validate the the deployment we receive has necessary fields set.
	if err := s.validateInput.do(deployment); err != nil {
		return nil, err
	}

	// Add/Update/Remove the deployment from persistence depending on the deployment action.
	if err := s.persistDeployment.do(action, deployment); err != nil {
		return nil, err
	}

	// Update secret service.
	if err := secretDataGraph.Singleton().ProcessDeploymentEvent(action, deployment); err != nil {
		return nil, err
	}

	// Process the deployment (enrichment, alert generation, enforcement action generation.)
	resp := s.createResponse.do(action, deployment)
	return resp, nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runGeneralPipeline(action v1.ResourceAction, deployment *v1.Deployment) (*v1.SensorEventResponse, error) {
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

	// Process the deployment (enrichment, alert generation, enforcement action generation.)
	resp := s.createResponse.do(action, deployment)

	// We want to persist the images from the deployment in the deployment after processing (create response)
	// TODO(rs): We should map out how images are updated in the pipeline so we don't do more writes than needed.
	s.updateImages.do(deployment)

	if err := secretDataGraph.Singleton().ProcessDeploymentEvent(action, deployment); err != nil {
		return nil, err
	}

	return resp, nil
}
