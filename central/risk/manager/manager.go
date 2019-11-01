package manager

import (
	"context"
	"time"

	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/metrics"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	deploymentScorer "github.com/stackrox/rox/central/risk/scorer/deployment"
	imageScorer "github.com/stackrox/rox/central/risk/scorer/image"
	imageComponentScorer "github.com/stackrox/rox/central/risk/scorer/image_component"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	log                = logging.LoggerForModule()
	riskReprocessorCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment, resources.Image, resources.Risk),
		))
	// Used for scorer.score() as the different Multipliers which will eventually use this context will require different permissions
	allAccessCtx = sac.WithAllAccess(context.Background())
)

// Manager manages changes to the risk of the deployments
type Manager interface {
	ReprocessDeploymentRisk(deployment *storage.Deployment)
	ReprocessDeploymentRiskWithImages(deployment *storage.Deployment, images []*storage.Image)
	ReprocessImageRisk(image *storage.Image)
	ReprocessImageComponentRisk(imageComponent *storage.EmbeddedImageScanComponent)
}

type managerImpl struct {
	deploymentStorage    deploymentDS.DataStore
	riskStorage          riskDS.DataStore
	deploymentScorer     deploymentScorer.Scorer
	imageScorer          imageScorer.Scorer
	imageComponentScorer imageComponentScorer.Scorer
}

// New returns a new manager
func New(deploymentStorage deploymentDS.DataStore,
	riskStorage riskDS.DataStore,
	deploymentScorer deploymentScorer.Scorer,
	imageScorer imageScorer.Scorer,
	imageComponentScorer imageComponentScorer.Scorer) (Manager, error) {
	m := &managerImpl{
		deploymentStorage:    deploymentStorage,
		riskStorage:          riskStorage,
		deploymentScorer:     deploymentScorer,
		imageScorer:          imageScorer,
		imageComponentScorer: imageComponentScorer,
	}
	return m, nil
}

// ReprocessDeploymentRisk will reprocess the passed deployments risk and save the results
func (e *managerImpl) ReprocessDeploymentRisk(deployment *storage.Deployment) {
	images, err := e.deploymentStorage.GetImagesForDeployment(riskReprocessorCtx, deployment)
	if err != nil {
		log.Errorf("error fetching images for deployment %s: %v", deployment.GetName(), err)
		return
	}
	e.ReprocessDeploymentRiskWithImages(deployment, images)

	// We want to compute and store risk for images when deployment risk is reprocessed.
	for _, image := range images {
		e.ReprocessImageRisk(image)
	}
}

// ReprocessDeploymentRiskWithImages will reprocess the passed deployments risk and save the results
func (e *managerImpl) ReprocessDeploymentRiskWithImages(deployment *storage.Deployment, images []*storage.Image) {
	defer metrics.ObserveRiskProcessingDuration(time.Now(), "Deployment")

	risk := e.deploymentScorer.Score(allAccessCtx, deployment, images)
	if risk == nil {
		return
	}

	if err := e.riskStorage.UpsertRisk(riskReprocessorCtx, risk); err != nil {
		log.Errorf("Error reprocessing risk for deployment %s: %v", deployment.GetName(), err)
	}
}

// ReprocessImageRisk will reprocess risk of the passed image and save the results.
func (e *managerImpl) ReprocessImageRisk(image *storage.Image) {
	defer metrics.ObserveRiskProcessingDuration(time.Now(), "Image")

	if !features.VulnMgmtUI.Enabled() {
		return
	}

	risk := e.imageScorer.Score(allAccessCtx, image)
	if risk == nil {
		return
	}

	if err := e.riskStorage.UpsertRisk(riskReprocessorCtx, risk); err != nil {
		log.Errorf("Error reprocessing risk for image %s: %v", image.GetName(), err)
	}

	// We want to compute and store risk for image components when image risk is reprocessed.
	for _, component := range image.GetScan().GetComponents() {
		e.ReprocessImageComponentRisk(component)
	}
}

// ReprocessImageComponentRisk will reprocess risk of image components and save the results.
// Image Component ID is generated as <component_name>:<component_version>
func (e *managerImpl) ReprocessImageComponentRisk(imageComponent *storage.EmbeddedImageScanComponent) {
	defer metrics.ObserveRiskProcessingDuration(time.Now(), "ImageComponent")

	if !features.VulnMgmtUI.Enabled() {
		return
	}

	risk := e.imageComponentScorer.Score(allAccessCtx, imageComponent)
	if risk == nil {
		return
	}

	if err := e.riskStorage.UpsertRisk(riskReprocessorCtx, risk); err != nil {
		log.Errorf("Error reprocessing risk for image component %s v%s: %v", imageComponent.GetName(), imageComponent.GetVersion(), err)
	}
}
