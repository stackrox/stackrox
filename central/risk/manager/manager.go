package manager

import (
	"context"
	"time"

	"github.com/pkg/errors"
	acUpdater "github.com/stackrox/rox/central/activecomponent/updater"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	iiStore "github.com/stackrox/rox/central/imageintegration/store"
	imageV2DS "github.com/stackrox/rox/central/imagev2/datastore"
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
	"github.com/stackrox/rox/pkg/protocompat"
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
	// TODO(ROX-30117): Remove CalculateRiskAndUpsertImage after ImageV2 model is fully rolled out
	CalculateRiskAndUpsertImage(image *storage.Image) error
	CalculateRiskAndUpsertImageV2(image *storage.ImageV2) error
	CalculateRiskAndUpsertNode(node *storage.Node) error

	// User ranking adjustment methods
	ChangeDeploymentRiskPosition(ctx context.Context, deploymentID string, moveUp bool) (*storage.Risk, error)
	ResetDeploymentRisk(ctx context.Context, deploymentID string) (*storage.Risk, error)
}

type managerImpl struct {
	deploymentStorage deploymentDS.DataStore
	nodeStorage       nodeDS.DataStore
	imageStorage      imageDS.DataStore
	imageV2Storage    imageV2DS.DataStore
	riskStorage       riskDS.DataStore

	deploymentScorer     deploymentScorer.Scorer
	nodeScorer           nodeScorer.Scorer
	imageScorer          imageScorer.Scorer
	imageComponentScorer componentScorer.ImageScorer
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
	imageV2Storage imageV2DS.DataStore,
	riskStorage riskDS.DataStore,
	nodeScorer nodeScorer.Scorer,
	nodeComponentScorer componentScorer.Scorer,
	deploymentScorer deploymentScorer.Scorer,
	imageScorer imageScorer.Scorer,
	imageComponentScorer componentScorer.ImageScorer,
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
		imageV2Storage:    imageV2Storage,
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

	// Always update deployment risk fields, even if score hasn't changed
	// This ensures effective_risk_score is populated for existing deployments
	deployment.RiskScore = risk.GetScore()
	deployment.EffectiveRiskScore = GetEffectiveScore(risk)

	// Set priority based on effective risk score (adjusted or ML score)
	// Priority is stored as int64 but represents a score, so multiply by 1000 to preserve precision
	deployment.Priority = int64(deployment.EffectiveRiskScore * 1000)

	// Only update namespace/cluster risk aggregations if score changed
	if oldScore != risk.GetScore() {
		e.updateNamespaceRisk(deployment.GetNamespaceId(), oldScore, risk.GetScore())
		e.updateClusterRisk(deployment.GetClusterId(), oldScore, risk.GetScore())
	}

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

	node.RiskScore = risk.GetScore()
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
	for index, component := range image.GetScan().GetComponents() {
		e.reprocessImageComponentRisk(component, image.GetScan().GetOperatingSystem(), image.GetId(), index)
	}

	image.RiskScore = risk.GetScore()
	return nil
}

// CalculateRiskAndUpsertImage will reprocess risk of the passed image and save the results.
func (e *managerImpl) CalculateRiskAndUpsertImage(image *storage.Image) error {
	if skip, err := e.skipImageUpsert(image); skip || err != nil {
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
	if features.ScannerV4.Enabled() && !scannedByScannerV4(img.GetScan().GetDataSource()) && e.scannedByClairify(img.GetScan().GetDataSource()) {
		// This image was scanned by the old Clairify scanner, we do not want to
		// overwrite an existing Scanner V4 scan in the database (if it exists).
		existingImg, exists, err := e.imageStorage.GetImage(riskReprocessorCtx, img.GetId())
		if err != nil {
			return false, err
		}
		if exists && scannedByScannerV4(existingImg.GetScan().GetDataSource()) {
			// Note: This image will not have `RiskScore` fields populated because
			// risk scores are heavily tied to upserting into Central DB and
			// this image is not being upserted.
			log.Warnw("Cannot overwrite Scanner V4 scan already in DB with Clairify scan and cannot calculate risk scores", logging.ImageName(img.GetName().GetFullName()), logging.ImageID(img.GetId()))
			return true, nil
		}
	}

	return false, nil
}

// scannedByClairify returns true if an image was scanned by the Clairify scanner, false otherwise.
func (e *managerImpl) scannedByClairify(dataSrc *storage.DataSource) bool {
	if dataSrc == nil {
		return false
	}

	for _, scanner := range e.iiSet.ScannerSet().GetAll() {
		if scanner.GetScanner().Type() == scannerTypes.Clairify && scanner.DataSource().GetId() == dataSrc.GetId() {
			return true
		}
	}

	return false
}

// scannedByScannerV4 returns true if an image was scanned by the Scanner V4 scanner, false otherwise.
func scannedByScannerV4(dataSrc *storage.DataSource) bool {
	return dataSrc.GetId() == iiStore.DefaultScannerV4Integration.GetId()
}

func (e *managerImpl) calculateAndUpsertImageV2Risk(image *storage.ImageV2) error {
	risk := e.imageScorer.ScoreV2(allAccessCtx, image)
	if risk == nil {
		return nil
	}

	if err := e.riskStorage.UpsertRisk(riskReprocessorCtx, risk); err != nil {
		return errors.Wrapf(err, "upserting risk for image %s", image.GetName().GetFullName())
	}

	// We want to compute and store risk for image components when image risk is reprocessed.
	for index, component := range image.GetScan().GetComponents() {
		e.reprocessImageComponentRisk(component, image.GetScan().GetOperatingSystem(), image.GetId(), index)
	}

	image.RiskScore = risk.GetScore()
	return nil
}

// CalculateRiskAndUpsertImageV2 will reprocess risk of the passed image and save the results.
func (e *managerImpl) CalculateRiskAndUpsertImageV2(image *storage.ImageV2) error {
	if skip, err := e.skipImageV2Upsert(image); err != nil {
		return err
	} else if skip {
		return nil
	}

	defer metrics.ObserveRiskProcessingDuration(time.Now(), "Image")

	if err := e.calculateAndUpsertImageV2Risk(image); err != nil {
		return errors.Wrapf(err, "calculating risk for image %s", image.GetName().GetFullName())
	}

	if err := e.imageV2Storage.UpsertImage(riskReprocessorCtx, image); err != nil {
		return errors.Wrapf(err, "upserting image %s", image.GetName().GetFullName())
	}
	return nil
}

func (e *managerImpl) skipImageV2Upsert(img *storage.ImageV2) (bool, error) {
	if features.ScannerV4.Enabled() && !scannedByScannerV4(img.GetScan().GetDataSource()) && e.scannedByClairify(img.GetScan().GetDataSource()) {
		// This image was scanned by the old Clairify scanner, we do not want to
		// overwrite an existing Scanner V4 scan in the database (if it exists).
		existingImg, exists, err := e.imageV2Storage.GetImage(riskReprocessorCtx, img.GetId())
		if err != nil {
			return false, err
		}
		if exists && scannedByScannerV4(existingImg.GetScan().GetDataSource()) {
			// Note: This image will not have `RiskScore` fields populated because
			// risk scores are heavily tied to upserting into Central DB and
			// this image is not being upserted.
			log.Warnw("Cannot overwrite Scanner V4 scan already in DB with Clairify scan and cannot calculate risk scores", logging.ImageName(img.GetName().GetFullName()), logging.ImageID(img.GetId()))
			return true, nil
		}
	}

	return false, nil
}

// reprocessImageComponentRisk will reprocess risk of image components and save the results.
// Image Component ID is generated as <component_name>:<component_version>
func (e *managerImpl) reprocessImageComponentRisk(imageComponent *storage.EmbeddedImageScanComponent, os string, imageID string, componentIndex int) {
	defer metrics.ObserveRiskProcessingDuration(time.Now(), "ImageComponent")

	risk := e.imageComponentScorer.Score(allAccessCtx, scancomponent.NewFromImageComponent(imageComponent), os, imageComponent, imageID, componentIndex)
	if risk == nil {
		return
	}

	var oldScore float32
	if features.FlattenCVEData.Enabled() {
		oldScore = e.imageComponentRanker.GetScoreForID(
			scancomponent.ComponentIDV2(imageComponent, imageID, componentIndex))
	} else {
		oldScore = e.imageComponentRanker.GetScoreForID(
			scancomponent.ComponentID(imageComponent.GetName(), imageComponent.GetVersion(), os))
	}

	// Image component risk results are currently unused so if the score is the same then no need to upsert
	if risk.GetScore() == oldScore {
		return
	}

	if err := e.riskStorage.UpsertRisk(riskReprocessorCtx, risk); err != nil {
		log.Errorf("Error reprocessing risk for image component %s %s: %v", imageComponent.GetName(), imageComponent.GetVersion(), err)
	}

	imageComponent.RiskScore = risk.GetScore()
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

	nodeComponent.RiskScore = risk.GetScore()
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

// ResetDeploymentRisk removes user ranking adjustments and returns to the original ML-calculated score.
func (e *managerImpl) ResetDeploymentRisk(ctx context.Context, deploymentID string) (*storage.Risk, error) {
	// Get the current risk
	risk, exists, err := e.riskStorage.GetRisk(ctx, deploymentID, storage.RiskSubjectType_DEPLOYMENT)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get risk for deployment %s", deploymentID)
	}
	if !exists {
		return nil, errors.Errorf("risk not found for deployment %s", deploymentID)
	}

	// Clear the user ranking adjustment
	risk.UserRankingAdjustment = nil

	// Save the updated risk
	if err := e.riskStorage.UpsertRisk(ctx, risk); err != nil {
		return nil, errors.Wrapf(err, "failed to reset risk for deployment %s", deploymentID)
	}

	// Reset the deployment's priority field to the original ML score
	deployment, exists, err := e.deploymentStorage.GetDeployment(ctx, deploymentID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get deployment %s to reset priority", deploymentID)
	}
	if !exists {
		return nil, errors.Errorf("deployment %s not found", deploymentID)
	}

	// Reset effective risk score to the original ML score
	deployment.EffectiveRiskScore = risk.GetScore()

	// Set priority based on effective risk score for consistency
	deployment.Priority = int64(deployment.EffectiveRiskScore * 1000)

	if err := e.deploymentStorage.UpsertDeployment(ctx, deployment); err != nil {
		return nil, errors.Wrapf(err, "failed to update deployment %s priority", deploymentID)
	}

	log.Infof("Reset user ranking adjustment for deployment %s, priority reset to %d, effective_risk_score %.2f", deploymentID, deployment.Priority, deployment.EffectiveRiskScore)
	return risk, nil
}

// ChangeDeploymentRiskPosition adjusts a deployment's risk ranking by moving it
// up or down in the ranking. It places the deployment midway between its current
// position and the next adjacent deployment.
func (e *managerImpl) ChangeDeploymentRiskPosition(ctx context.Context, deploymentID string, moveUp bool) (*storage.Risk, error) {
	// Get all deployment risks in user's scope
	allRisks, err := e.riskStorage.GetDeploymentsInUserScope(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get deployments in user scope")
	}

	if len(allRisks) == 0 {
		return nil, errors.New("no deployments found in user scope")
	}

	// Find the target deployment
	var targetRisk *storage.Risk
	for _, risk := range allRisks {
		if risk.GetSubject().GetId() == deploymentID {
			targetRisk = risk
			break
		}
	}

	if targetRisk == nil {
		return nil, errors.Errorf("deployment %s not found in user scope", deploymentID)
	}

	// Sort risks by effective score (descending)
	sortedRisks := SortRisksByEffectiveScore(allRisks)

	// Find current position
	currentIndex := FindDeploymentIndex(sortedRisks, deploymentID)
	if currentIndex == -1 {
		return nil, errors.Errorf("deployment %s not found in sorted list", deploymentID)
	}

	// Get current effective score
	currentScore := GetEffectiveScore(targetRisk)

	// Calculate new score
	newScore := CalculatePositionChangeScore(currentScore, sortedRisks, currentIndex, moveUp)

	// If score didn't change, it's a no-op (at boundary)
	if newScore == currentScore {
		direction := "up"
		if !moveUp {
			direction = "down"
		}
		log.Infof("Deployment %s position change %s is a no-op (at boundary)", deploymentID, direction)
		return targetRisk, nil
	}

	// Create or update user ranking adjustment
	if targetRisk.GetUserRankingAdjustment() == nil {
		targetRisk.UserRankingAdjustment = &storage.UserRankingAdjustment{}
	}

	adj := targetRisk.GetUserRankingAdjustment()
	adj.AdjustedScore = newScore
	adj.LastAdjusted = protocompat.TimestampNow()

	// Get user ID from context (if available via authn)
	// For now, we'll use a placeholder - this should be extracted from the context
	adj.LastAdjustedBy = "user" // TODO: Extract from authn context

	// Save the updated risk
	if err := e.riskStorage.UpsertRisk(ctx, targetRisk); err != nil {
		return nil, errors.Wrapf(err, "failed to update risk for deployment %s", deploymentID)
	}

	// Update the deployment's priority field to reflect the adjusted score
	// This ensures sorting by "Deployment Risk Priority" uses the user-adjusted score
	deployment, exists, err := e.deploymentStorage.GetDeployment(ctx, deploymentID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get deployment %s to update priority", deploymentID)
	}
	if !exists {
		return nil, errors.Errorf("deployment %s not found", deploymentID)
	}

	// Store the effective risk score on the deployment
	deployment.EffectiveRiskScore = newScore

	// Set priority based on effective risk score for consistency
	// Priority is stored as int64 but represents a score, so we multiply by 1000 to preserve precision
	deployment.Priority = int64(deployment.GetEffectiveRiskScore() * 1000)

	if err := e.deploymentStorage.UpsertDeployment(ctx, deployment); err != nil {
		return nil, errors.Wrapf(err, "failed to update deployment %s priority", deploymentID)
	}

	direction := "up"
	if !moveUp {
		direction = "down"
	}
	log.Infof("Deployment %s moved %s: score %.2f -> %.2f, priority updated to %d, effective_risk_score %.2f", deploymentID, direction, currentScore, newScore, deployment.Priority, deployment.GetEffectiveRiskScore())

	return targetRisk, nil
}
