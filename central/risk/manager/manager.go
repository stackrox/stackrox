package manager

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	pkgImgComponent "github.com/stackrox/rox/central/imagecomponent"
	imageComponentDS "github.com/stackrox/rox/central/imagecomponent/datastore"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/ranking"
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
//go:generate mockgen-wrapper Manager
type Manager interface {
	ReprocessDeploymentRisk(deployment *storage.Deployment)
	CalculateRiskAndUpsertImage(image *storage.Image) error
}

type managerImpl struct {
	deploymentStorage     deploymentDS.DataStore
	imageStorage          imageDS.DataStore
	imageComponentStorage imageComponentDS.DataStore
	riskStorage           riskDS.DataStore

	deploymentScorer     deploymentScorer.Scorer
	imageScorer          imageScorer.Scorer
	imageComponentScorer imageComponentScorer.Scorer

	clusterRanker        *ranking.Ranker
	nsRanker             *ranking.Ranker
	deploymentRanker     *ranking.Ranker
	imageRanker          *ranking.Ranker
	imageComponentRanker *ranking.Ranker
}

// New returns a new manager
func New(deploymentStorage deploymentDS.DataStore,
	imageStorage imageDS.DataStore,
	imageComponentStorage imageComponentDS.DataStore,
	riskStorage riskDS.DataStore,
	deploymentScorer deploymentScorer.Scorer,
	imageScorer imageScorer.Scorer,
	imageComponentScorer imageComponentScorer.Scorer,
	clusterRanker *ranking.Ranker,
	nsRanker *ranking.Ranker,
	deploymentRanker *ranking.Ranker,
	imageRanker *ranking.Ranker,
	imageComponentRanker *ranking.Ranker) (Manager, error) {
	m := &managerImpl{
		deploymentStorage:     deploymentStorage,
		imageStorage:          imageStorage,
		imageComponentStorage: imageComponentStorage,
		riskStorage:           riskStorage,

		deploymentScorer:     deploymentScorer,
		imageScorer:          imageScorer,
		imageComponentScorer: imageComponentScorer,

		clusterRanker:        clusterRanker,
		nsRanker:             nsRanker,
		deploymentRanker:     deploymentRanker,
		imageRanker:          imageRanker,
		imageComponentRanker: imageComponentRanker,
	}
	return m, nil
}

// ReprocessDeploymentRisk will reprocess the passed deployments risk and save the results
func (e *managerImpl) ReprocessDeploymentRisk(deployment *storage.Deployment) {
	defer metrics.ObserveRiskProcessingDuration(time.Now(), "Deployment")

	oldRisk, exists, err := e.riskStorage.GetRisk(allAccessCtx, deployment.GetId(), storage.RiskSubjectType_DEPLOYMENT)
	if err != nil {
		log.Errorf("error getting risk for deployment %s: %v", deployment.GetName(), err)
	}

	// Get Image Risk
	imageRisks := make([]*storage.Risk, 0, len(deployment.GetContainers()))
	for _, container := range deployment.GetContainers() {
		if imgID := container.GetImage().GetId(); imgID != "" {
			risk, exists, err := e.riskStorage.GetRisk(allAccessCtx, imgID, storage.RiskSubjectType_IMAGE)
			if err != nil {
				log.Errorf("error getting risk for image %s: %v", imgID, err)
				continue
			}
			if !exists {
				continue
			}
			imageRisks = append(imageRisks, risk)
		}
	}

	risk := e.deploymentScorer.Score(allAccessCtx, deployment, imageRisks)
	if risk == nil {
		return
	}

	// No need to insert if it hasn't changed
	if exists && proto.Equal(oldRisk, risk) {
		return
	}

	oldScore := float32(-1)
	if exists {
		oldScore = oldRisk.GetScore()
	}

	if err := e.riskStorage.UpsertRisk(riskReprocessorCtx, risk); err != nil {
		log.Errorf("Error reprocessing risk for deployment %s: %v", deployment.GetName(), err)
	}

	if oldScore == risk.GetScore() {
		return
	}

	e.updateNamespaceRisk(deployment.GetNamespaceId(), oldScore, risk.GetScore())
	e.updateClusterRisk(deployment.GetClusterId(), oldScore, risk.GetScore())

	deployment.RiskScore = risk.Score
	if err := e.deploymentStorage.UpsertDeployment(riskReprocessorCtx, deployment); err != nil {
		log.Errorf("error upserting deployment: %v", err)
	}
}

func (e *managerImpl) calculateAndUpsertImageRisk(image *storage.Image) error {
	risk := e.imageScorer.Score(allAccessCtx, image)
	if risk == nil {
		return nil
	}

	if err := e.riskStorage.UpsertRisk(riskReprocessorCtx, risk); err != nil {
		return errors.Wrapf(err, "upserting risk for image %s", image.GetName().GetFullName())
	}

	// We want to compute and store risk for image components when image risk is reprocessed.
	for _, component := range image.GetScan().GetComponents() {
		e.reprocessImageComponentRisk(component)
	}

	image.RiskScore = risk.Score
	return nil
}

// ReprocessImageRisk will reprocess risk of the passed image and save the results.
func (e *managerImpl) CalculateRiskAndUpsertImage(image *storage.Image) error {
	defer metrics.ObserveRiskProcessingDuration(time.Now(), "Image")

	if !features.VulnMgmtUI.Enabled() {
		return nil
	}

	if err := e.calculateAndUpsertImageRisk(image); err != nil {
		return errors.Wrapf(err, "calculating risk for image %s", image.GetName().GetFullName())
	}

	if err := e.imageStorage.UpsertImage(riskReprocessorCtx, image); err != nil {
		return errors.Wrapf(err, "upserting image %s", image.GetName().GetFullName())
	}
	return nil
}

// reprocessImageComponentRisk will reprocess risk of image components and save the results.
// Image Component ID is generated as <component_name>:<component_version>
func (e *managerImpl) reprocessImageComponentRisk(imageComponent *storage.EmbeddedImageScanComponent) {
	defer metrics.ObserveRiskProcessingDuration(time.Now(), "ImageComponent")

	if !features.VulnMgmtUI.Enabled() {
		return
	}

	risk := e.imageComponentScorer.Score(allAccessCtx, imageComponent)
	if risk == nil {
		return
	}

	oldScore := e.imageComponentRanker.GetScoreForID(
		pkgImgComponent.ComponentID{
			Name:    imageComponent.GetName(),
			Version: imageComponent.GetVersion(),
		}.ToString())
	if err := e.riskStorage.UpsertRisk(riskReprocessorCtx, risk); err != nil {
		log.Errorf("Error reprocessing risk for image component %s v%s: %v", imageComponent.GetName(), imageComponent.GetVersion(), err)
	}

	if !features.Dackbox.Enabled() {
		return
	}

	if oldScore == risk.GetScore() {
		return
	}

	imageComponent.RiskScore = risk.Score
	// skip direct upsert here since it is handled during image upsert
}

func (e *managerImpl) updateNamespaceRisk(nsID string, oldDeploymentScore float32, newDeploymentScore float32) {
	oldNSRiskScore := e.nsRanker.GetScoreForID(nsID)
	e.nsRanker.Add(nsID, oldNSRiskScore-oldDeploymentScore+newDeploymentScore)
}

func (e *managerImpl) updateClusterRisk(clusterID string, oldDeploymentScore float32, newDeploymentScore float32) {
	oldClusterRiskScore := e.nsRanker.GetScoreForID(clusterID)
	e.clusterRanker.Add(clusterID, oldClusterRiskScore-oldDeploymentScore+newDeploymentScore)
}
