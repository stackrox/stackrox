package pipeline

import (
	"context"

	clusterDataStore "bitbucket.org/stack-rox/apollo/central/cluster/datastore"
	deploymentDataStore "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	"bitbucket.org/stack-rox/apollo/central/detection"
	globaldb "bitbucket.org/stack-rox/apollo/central/globaldb/singletons"
	globalindex "bitbucket.org/stack-rox/apollo/central/globalindex/singletons"
	imageDataStore "bitbucket.org/stack-rox/apollo/central/image/datastore"
	secretUpdate "bitbucket.org/stack-rox/apollo/central/secret/datagraph/deploymentevent"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Template design pattern. We define control flow here and defer logic to subclasses.
//////////////////////////////////////////////////////////////////////////////////////

// Pipeline represents the processing applied to a DeploymentEvent to produce a response.
type Pipeline interface {
	Run(event *v1.DeploymentEvent) (*v1.DeploymentEventResponse, error)
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(ctx context.Context, clusters clusterDataStore.DataStore, deployments deploymentDataStore.DataStore, images imageDataStore.DataStore, detector *detection.Detector) Pipeline {
	return &pipelineImpl{
		validateInput:     newValidateInput(),
		clusterEnrichment: newClusterEnrichment(ctx, clusters),
		updateImages:      newUpdateImages(images),
		persistDeployment: newPersistDeployment(deployments),
		createResponse:    newCreateResponse(detector.ProcessDeploymentEvent),
		persistImages:     newPersistImages(images),
	}
}

type pipelineImpl struct {
	// pipeline stages.
	validateInput     *validateInputImpl
	clusterEnrichment *clusterEnrichmentImpl
	updateImages      *updateImagesImpl
	persistDeployment *persistDeploymentImpl
	createResponse    *createResponseImpl
	persistImages     *persistImagesImpl
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(event *v1.DeploymentEvent) (*v1.DeploymentEventResponse, error) {
	switch event.GetAction() {
	case v1.ResourceAction_REMOVE_RESOURCE:
		return s.runRemovePipeline(event)
	default:
		return s.runGeneralPipeline(event)
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runRemovePipeline(event *v1.DeploymentEvent) (*v1.DeploymentEventResponse, error) {
	// Validate the the event we receive has necessary fields set.
	if err := s.validateInput.do(event); err != nil {
		return nil, err
	}

	// Add/Update/Remove the deployment from persistence depending on the event action.
	if err := s.persistDeployment.do(event); err != nil {
		return nil, err
	}

	// Update secret service.
	if err := secretUpdate.ProcessDeploymentEvent(globaldb.GetGlobalDB(), globalindex.GetGlobalIndex(), event); err != nil {
		return nil, err
	}

	// Process the event (enrichment, alert generation, enforcement action generation.)
	resp := s.createResponse.do(event)
	return resp, nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runGeneralPipeline(event *v1.DeploymentEvent) (*v1.DeploymentEventResponse, error) {
	// Validate the the event we receive has necessary fields set.
	if err := s.validateInput.do(event); err != nil {
		return nil, err
	}

	// Fill in cluster information.
	if err := s.clusterEnrichment.do(event.GetDeployment()); err != nil {
		log.Errorf("Couldn't get cluster identity: %s", err)
	}

	// Add/Update/Remove the deployment from persistence depending on the event action.
	if err := s.persistDeployment.do(event); err != nil {
		return nil, err
	}

	// Update the deployments images with the latest version from storage.
	s.updateImages.do(event.GetDeployment())

	// Process the event (enrichment, alert generation, enforcement action generation.)
	resp := s.createResponse.do(event)

	// We want to persist the images from the deployment in the event after processing (create response)
	// TODO(rs): We should map out how images are updated in the pipeline so we don't do more writes than needed.
	s.persistImages.do(event)

	// Update secret service.
	if err := secretUpdate.ProcessDeploymentEvent(globaldb.GetGlobalDB(), globalindex.GetGlobalIndex(), event); err != nil {
		return nil, err
	}

	return resp, nil
}
