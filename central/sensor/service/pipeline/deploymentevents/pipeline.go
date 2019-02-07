package deploymentevents

import (
	"github.com/gogo/protobuf/proto"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/lifecycle"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/networkpolicies/graph"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/enforcers"
	"github.com/stackrox/rox/pkg/errorhelpers"
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

	graphEvaluator graph.Evaluator

	reconcileStore reconciliation.Store
}

func (s *pipelineImpl) Reconcile(clusterID string) error {
	defer s.reconcileStore.Close()

	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	results, err := s.deployments.Search(query)
	if err != nil {
		return err
	}

	idsToDelete := search.ResultsToIDSet(results).Difference(s.reconcileStore.GetSet()).AsSlice()
	if len(idsToDelete) > 0 {
		log.Infof("Deleting deployments %+v as a part of reconciliation", idsToDelete)
	}
	errList := errorhelpers.NewErrorList("Deployment reconciliation")
	for _, id := range idsToDelete {
		_, err := s.runRemovePipeline(central.ResourceAction_REMOVE_RESOURCE, &storage.Deployment{Id: id})
		errList.AddError(err)
	}
	return errList.ToError()
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetDeployment() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(msg *central.MsgFromSensor, injector pipeline.MsgInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.Deployment)

	event := msg.GetEvent()
	deployment := event.GetDeployment()
	deployment.ClusterId = event.GetClusterId()

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
			injected := injector.InjectMessage(&central.MsgToSensor{
				Msg: &central.MsgToSensor_Enforcement{
					Enforcement: resp,
				},
			})
			if !injected {
				log.Errorf("Failed to inject enforcement action %s", proto.MarshalTextString(resp))
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

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runGeneralPipeline(action central.ResourceAction, deployment *storage.Deployment) (*central.SensorEnforcement, error) {
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

func (s *pipelineImpl) OnFinish() {}
