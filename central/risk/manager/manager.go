package manager

import (
	"context"
	"time"

	"github.com/pkg/errors"
	acUpdater "github.com/stackrox/rox/central/activecomponent/updater"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	iiStore "github.com/stackrox/rox/central/imageintegration/store"
	"github.com/stackrox/rox/central/metrics"
	nodeDS "github.com/stackrox/rox/central/node/datastore"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	componentScorer "github.com/stackrox/rox/central/risk/scorer/component"
	deploymentScorer "github.com/stackrox/rox/central/risk/scorer/deployment"
	imageScorer "github.com/stackrox/rox/central/risk/scorer/image"
	nodeScorer "github.com/stackrox/rox/central/risk/scorer/node"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/integration"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/scancomponent"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
)

var (
	log                = logging.LoggerForModule()
	riskReprocessorCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment, resources.Image, resources.DeploymentExtension, resources.Node),
		))
	// Used for scorer.score() as the different Multipliers which will eventually use this context will require different permissions
	allAccessCtx = sac.WithAllAccess(context.Background())
)

// Manager manages changes to the risk of deployments and nodes
//
//go:generate mockgen-wrapper
type Manager interface {
	ReprocessDeploymentRisk(deployment *storage.Deployment)
	CalculateRiskAndUpsertImage(image *storage.Image) error
	CalculateRiskAndUpsertNode(node *storage.Node) error
}

type managerImpl struct {
	deploymentStorage deploymentDS.DataStore
	nodeStorage       nodeDS.DataStore
	imageStorage      imageDS.DataStore
	riskStorage       riskDS.DataStore

	deploymentScorer     deploymentScorer.Scorer
	nodeScorer           nodeScorer.Scorer
	imageScorer          imageScorer.Scorer
	imageComponentScorer componentScorer.Scorer
	nodeComponentScorer  componentScorer.Scorer

	clusterRanker        *ranking.Ranker
	nsRanker             *ranking.Ranker
	imageComponentRanker *ranking.Ranker
	nodeComponentRanker  *ranking.Ranker

	acUpdater acUpdater.Updater

	iiSet integration.Set
}

// New returns a new manager
func New(nodeStorage nodeDS.DataStore,
	deploymentStorage deploymentDS.DataStore,
	imageStorage imageDS.DataStore,
	riskStorage riskDS.DataStore,
	nodeScorer nodeScorer.Scorer,
	nodeComponentScorer componentScorer.Scorer,
	deploymentScorer deploymentScorer.Scorer,
	imageScorer imageScorer.Scorer,
	imageComponentScorer componentScorer.Scorer,
	clusterRanker *ranking.Ranker,
	nsRanker *ranking.Ranker,
	componentRanker *ranking.Ranker,
	nodeComponentRanker *ranking.Ranker,
	acUpdater acUpdater.Updater,
	iiSet integration.Set,
) Manager {
	m := &managerImpl{
		nodeStorage:       nodeStorage,
		deploymentStorage: deploymentStorage,
		imageStorage:      imageStorage,
		riskStorage:       riskStorage,

		nodeScorer:           nodeScorer,
		nodeComponentScorer:  nodeComponentScorer,
		deploymentScorer:     deploymentScorer,
		imageScorer:          imageScorer,
		imageComponentScorer: imageComponentScorer,

		clusterRanker:        clusterRanker,
		nsRanker:             nsRanker,
		imageComponentRanker: componentRanker,
		nodeComponentRanker:  nodeComponentRanker,
		acUpdater:            acUpdater,

		iiSet: iiSet,
	}
	return m
}

// ReprocessDeploymentRisk will reprocess the passed deployment's risk and save the results
func (e *managerImpl) ReprocessDeploymentRisk(deployment *storage.Deployment) {
	defer metrics.ObserveRiskProcessingDuration(time.Now(), "Deployment")

	oldRisk, exists, err := e.riskStorage.GetRiskForDeployment(allAccessCtx, deployment)
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
	if exists && oldRisk.EqualVT(risk) {
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

func (e *managerImpl) calculateAndUpsertNodeRisk(node *storage.Node) error {
	risk := e.nodeScorer.Score(allAccessCtx, node)
	if risk == nil {
		return nil
	}

	if err := e.riskStorage.UpsertRisk(riskReprocessorCtx, risk); err != nil {
		return errors.Wrapf(err, "upserting risk for node %s", node.GetName())
	}

	// We want to compute and store risk for node components when node risk is reprocessed.
	for _, c := range node.GetScan().GetComponents() {
		e.reprocessNodeComponentRisk(c, node.GetScan().GetOperatingSystem())
	}

	node.RiskScore = risk.Score
	return nil
}

func (e *managerImpl) CalculateRiskAndUpsertNode(node *storage.Node) error {
	defer metrics.ObserveRiskProcessingDuration(time.Now(), "Node")

	if err := e.calculateAndUpsertNodeRisk(node); err != nil {
		return errors.Wrapf(err, "calculating risk for node %s", nodeDS.NodeString(node))
	}

	// TODO: ROX-6235: Evaluate cluster risk.

	if err := e.nodeStorage.UpsertNode(riskReprocessorCtx, node); err != nil {
		return errors.Wrapf(err, "upserting node %s", nodeDS.NodeString(node))
	}
	return nil
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
		e.reprocessImageComponentRisk(component, image.GetScan().GetOperatingSystem())
	}

	image.RiskScore = risk.Score
	return nil
}

// CalculateRiskAndUpsertImage will reprocess risk of the passed image and save the results.
func (e *managerImpl) CalculateRiskAndUpsertImage(image *storage.Image) error {
	log.Info("In CalculateRiskAndUpsertImage")
	if skip, err := e.skipImageUpsert(image); skip || err != nil {
		log.Info("Skipping upsert")
		return err
	}

	defer metrics.ObserveRiskProcessingDuration(time.Now(), "Image")

	if err := e.calculateAndUpsertImageRisk(image); err != nil {
		return errors.Wrapf(err, "calculating risk for image %s", image.GetName().GetFullName())
	}

	if err := e.imageStorage.UpsertImage(riskReprocessorCtx, image); err != nil {
		return errors.Wrapf(err, "upserting image %s", image.GetName().GetFullName())
	}

	if err := e.acUpdater.PopulateExecutableCache(riskReprocessorCtx, image); err != nil {
		return errors.Wrapf(err, "populating executable cache for image %s", image.GetId())
	}
	return nil
}

// skipImageUpsert will return true if an image should not be upserted into the store.
func (e *managerImpl) skipImageUpsert(img *storage.Image) (bool, error) {
	if features.ScannerV4.Enabled() && !scannedByScannerV4(img) && e.scannedByClairify(img) {
		// This image was scanned by the old Clairify scanner, we do not want to
		// overwrite an existing Scanner V4 scan in the database (if it exists).
		existingImg, exists, err := e.imageStorage.GetImage(riskReprocessorCtx, img.GetId())
		if err != nil {
			return false, err
		}
		if exists && scannedByScannerV4(existingImg) {
			// Note: This image will not have `RiskScore` fields populated because
			// risk scores are heavily tied to upserting into Central DB and
			// this image is not being upserted.
			log.Warnw("Cannot overwrite Scanner V4 scan already in DB with Clairify scan and cannot calculate risk scores", logging.ImageName(img.GetName().GetFullName()))
			return true, nil
		}
	}

	return false, nil
}

// scannedByClairify returns true if an image was scanned by the Clairify scanner, false otherwise.
func (e *managerImpl) scannedByClairify(img *storage.Image) bool {
	if img.GetScan().GetDataSource() == nil {
		return false
	}

	for _, scanner := range e.iiSet.ScannerSet().GetAll() {
		if scanner.GetScanner().Type() == scannerTypes.Clairify && scanner.DataSource().GetId() == img.GetScan().GetDataSource().GetId() {
			return true
		}
	}

	return false
}

// scannedByScannerV4 returns true if an image was scanned by the Scanner V4 scanner, false otherwise.
func scannedByScannerV4(img *storage.Image) bool {
	return img.GetScan().GetDataSource().GetId() == iiStore.DefaultScannerV4Integration.GetId()
}

// reprocessImageComponentRisk will reprocess risk of image components and save the results.
// Image Component ID is generated as <component_name>:<component_version>
func (e *managerImpl) reprocessImageComponentRisk(imageComponent *storage.EmbeddedImageScanComponent, os string) {
	defer metrics.ObserveRiskProcessingDuration(time.Now(), "ImageComponent")

	risk := e.imageComponentScorer.Score(allAccessCtx, scancomponent.NewFromImageComponent(imageComponent), os)
	if risk == nil {
		return
	}

	oldScore := e.imageComponentRanker.GetScoreForID(
		scancomponent.ComponentID(imageComponent.GetName(), imageComponent.GetVersion(), os))

	// Image component risk results are currently unused so if the score is the same then no need to upsert
	if risk.GetScore() == oldScore {
		return
	}

	if err := e.riskStorage.UpsertRisk(riskReprocessorCtx, risk); err != nil {
		log.Errorf("Error reprocessing risk for image component %s %s: %v", imageComponent.GetName(), imageComponent.GetVersion(), err)
	}

	imageComponent.RiskScore = risk.Score
	// skip direct upsert here since it is handled during image upsert
}

// reprocessNodeComponentRisk will reprocess risk of node components and save the results.
// Node Component ID is generated as <component_name>:<component_version>
func (e *managerImpl) reprocessNodeComponentRisk(nodeComponent *storage.EmbeddedNodeScanComponent, os string) {
	defer metrics.ObserveRiskProcessingDuration(time.Now(), "NodeComponent")

	risk := e.nodeComponentScorer.Score(allAccessCtx, scancomponent.NewFromNodeComponent(nodeComponent), os)
	if risk == nil {
		return
	}

	oldScore := e.nodeComponentRanker.GetScoreForID(
		scancomponent.ComponentID(nodeComponent.GetName(), nodeComponent.GetVersion(), os))

	// Node component risk results are not currently used so if the score is the same then no need to upsert
	if risk.GetScore() == oldScore {
		return
	}

	if err := e.riskStorage.UpsertRisk(riskReprocessorCtx, risk); err != nil {
		log.Errorf("Error reprocessing risk for node component %s %s: %v", nodeComponent.GetName(), nodeComponent.GetVersion(), err)
	}

	nodeComponent.RiskScore = risk.Score
	// skip direct upsert here since it is handled during node upsert
}

func (e *managerImpl) updateNamespaceRisk(nsID string, oldDeploymentScore float32, newDeploymentScore float32) {
	oldNSRiskScore := e.nsRanker.GetScoreForID(nsID)
	e.nsRanker.Add(nsID, oldNSRiskScore-oldDeploymentScore+newDeploymentScore)
}

// TODO: ROX-6235: Account for node risk.
func (e *managerImpl) updateClusterRisk(clusterID string, oldDeploymentScore float32, newDeploymentScore float32) {
	oldClusterRiskScore := e.clusterRanker.GetScoreForID(clusterID)
	e.clusterRanker.Add(clusterID, oldClusterRiskScore-oldDeploymentScore+newDeploymentScore)
}
