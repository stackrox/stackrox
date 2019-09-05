package manager

import (
	"context"
	"time"

	"github.com/pkg/errors"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/metrics"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/central/risk/scorer"
	"github.com/stackrox/rox/central/role/resources"
	serviceAccDS "github.com/stackrox/rox/central/serviceaccount/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	pkgRisk "github.com/stackrox/rox/pkg/risk"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

var (
	log                = logging.LoggerForModule()
	riskReprocessorCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment, resources.Image, resources.Risk, resources.ServiceAccount),
		))
	// Used for scorer.score() as the different Multipliers which will eventually use this context will require different permissions
	allAccessCtx = sac.WithAllAccess(context.Background())
)

// Manager manages changes to the risk of the deployments
//go:generate mockgen-wrapper
type Manager interface {
	ReprocessDeploymentRisk(deploymentID string, riskIndicators ...pkgRisk.Indicator)
	ReprocessRiskForDeployments(deploymentIDs []string, riskIndicators ...pkgRisk.Indicator)
	ReprocessRiskForAllDeployments(riskIndicators ...pkgRisk.Indicator)
}

type managerImpl struct {
	deploymentStorage deploymentDS.DataStore
	riskStorage       riskDS.DataStore
	serviceAccStorage serviceAccDS.DataStore
	scorer            scorer.Scorer
}

// New returns a new manager
func New(
	deploymentStorage deploymentDS.DataStore,
	serviceAccStorage serviceAccDS.DataStore,
	riskStorage riskDS.DataStore,
	scorer scorer.Scorer) (Manager, error) {
	m := &managerImpl{
		deploymentStorage: deploymentStorage,
		riskStorage:       riskStorage,
		serviceAccStorage: serviceAccStorage,
		scorer:            scorer,
	}
	return m, nil
}

// ReprocessRiskForAllDeployments will reprocess risk for all deployments for given risk indicators.
// If no risk indicators are supplied, all risk indicators are processed.
func (e *managerImpl) ReprocessRiskForAllDeployments(riskIndicators ...pkgRisk.Indicator) {
	defer metrics.ObserveRiskProcessingDuration(time.Now())
	deployments, err := e.getDeployments()
	if err != nil {
		log.Error(err)
		return
	}
	for _, deployment := range deployments {
		e.reprocessRiskByIndicators(deployment, riskIndicators...)
	}
}

// ReprocessRiskForDeployments will reprocess risk for all deployments for given risk indicators.
// If no risk indicators are supplied, all risk indicators are processed.
func (e *managerImpl) ReprocessRiskForDeployments(deploymentIDs []string, riskIndicators ...pkgRisk.Indicator) {
	defer metrics.ObserveRiskProcessingDuration(time.Now())
	deployments, err := e.getDeployments(deploymentIDs...)
	if err != nil {
		log.Error(err)
		return
	}
	for _, deployment := range deployments {
		e.reprocessRiskByIndicators(deployment, riskIndicators...)
	}
}

// ReprocessDeploymentRisk will reprocess deployment's risk for given risk indicators.
// If no risk indicators are supplied, all risk indicators are processed.
func (e *managerImpl) ReprocessDeploymentRisk(deploymentID string, riskIndicators ...pkgRisk.Indicator) {
	defer metrics.ObserveRiskProcessingDuration(time.Now())
	deployments, err := e.getDeployments(deploymentID)
	if err != nil {
		log.Error(err)
		return
	}
	if len(deployments) == 0 {
		log.Errorf("error fetching deployment %s for risk reprocessing", deploymentID)
		return
	}
	e.reprocessRiskByIndicators(deployments[0], riskIndicators...)
}

func (e *managerImpl) getDeployments(deploymentIDs ...string) ([]*storage.Deployment, error) {
	query := search.NewQueryBuilder().AddStringsHighlighted(search.ClusterID, search.WildcardString)
	if len(deploymentIDs) > 0 {
		query = query.AddDocIDs(deploymentIDs...)
	}
	deployments, err := e.deploymentStorage.SearchRawDeployments(riskReprocessorCtx, query.ProtoQuery())
	if err != nil {
		return nil, errors.Wrapf(err, "error getting deployments for risk reprocessing")
	}
	return deployments, nil
}

func (e *managerImpl) reprocessRiskByIndicators(deployment *storage.Deployment, riskIndicators ...pkgRisk.Indicator) {
	if len(riskIndicators) == 0 {
		for _, v := range pkgRisk.AllIndicatorMap {
			riskIndicators = append(riskIndicators, v)
		}
	}

	deploymentRiskID, err := pkgRisk.GetID(deployment.GetId(), storage.RiskEntityType_DEPLOYMENT)
	if err != nil {
		log.Error(err)
		return
	}

	images, err := e.deploymentStorage.GetImagesForDeployment(riskReprocessorCtx, deployment)
	if err != nil {
		log.Errorf("error fetching images for deployment %s: %v", deployment.GetId(), err)
		return
	}

	processedImage := set.NewStringSet()
	for _, image := range images {
		if processedImage.Contains(image.GetId()) {
			continue
		}
		imageRisk := e.reprocessImageRisk(image, riskIndicators...)
		if imageRisk == nil {
			continue
		}
		processedImage.Add(image.GetId())
		e.riskStorage.AddRiskDependencies(deploymentRiskID, imageRisk.GetId())
		e.mergeAndUpsertRisk(imageRisk)
	}

	serviceAccRisk := e.reprocessServiceAccRisk(deployment, riskIndicators...)
	if serviceAccRisk != nil {
		e.riskStorage.AddRiskDependencies(deploymentRiskID, serviceAccRisk.GetId())
		e.mergeAndUpsertRisk(serviceAccRisk)
	}

	// Score deployment after dependencies are processed.
	deploymentRisk := e.reprocessDeploymentRisk(deployment, riskIndicators...)
	e.mergeAndUpsertRisk(deploymentRisk)
}

func (e *managerImpl) reprocessImageRisk(image *storage.Image, riskIndicators ...pkgRisk.Indicator) *storage.Risk {
	// Images may not have ID when deployment is launched.
	if image.GetId() == "" {
		return nil
	}
	risk := e.scorer.Score(allAccessCtx, image, riskIndicators...)
	return e.mergeAndUpsertRisk(risk)
}

func (e *managerImpl) reprocessServiceAccRisk(deployment *storage.Deployment, riskIndicators ...pkgRisk.Indicator) *storage.Risk {
	serviceAcc, err := e.serviceAccStorage.SearchRawServiceAccounts(riskReprocessorCtx,
		search.NewQueryBuilder().AddExactMatches(search.ServiceAccountName, deployment.GetServiceAccount()).
			AddExactMatches(search.Namespace, deployment.GetNamespace()).
			AddExactMatches(search.ClusterID, deployment.GetClusterId()).ProtoQuery())
	if err != nil {
		log.Errorf("error fetching service account for deployment %s: %v", deployment.GetId(), err)
		return nil
	}
	if len(serviceAcc) == 0 {
		return nil
	}
	// We need to ensure that only service account risk indicators are processed.
	// Otherwise, since we are passing deployment, multiplier interface will compute all indicators that accept deployment.
	filtered := riskIndicators[:0]
	for _, indicator := range riskIndicators {
		if indicator.EntityAppliedTo != storage.RiskEntityType_SERVICEACCOUNT {
			continue
		}
		filtered = append(filtered, indicator)
	}
	risk := e.scorer.Score(allAccessCtx, deployment, filtered...)
	if risk == nil {
		return nil
	}

	srvAccRisk := pkgRisk.BuildRiskProtoForServiceAccount(serviceAcc[0])
	srvAccRisk.Results = risk.Results
	srvAccRisk.Score = risk.Score

	return srvAccRisk
}

func (e *managerImpl) reprocessDeploymentRisk(deployment *storage.Deployment, riskIndicators ...pkgRisk.Indicator) *storage.Risk {
	filtered := riskIndicators[:0]
	for _, indicator := range riskIndicators {
		if indicator.EntityAppliedTo != storage.RiskEntityType_DEPLOYMENT {
			continue
		}
		filtered = append(filtered, indicator)
	}
	return e.scorer.Score(allAccessCtx, deployment, filtered...)
}

func (e *managerImpl) mergeAndUpsertRisk(risk *storage.Risk) *storage.Risk {
	if risk == nil {
		return nil
	}
	oldRisk, found, err := e.riskStorage.GetRisk(riskReprocessorCtx, risk.GetEntity().GetId(), risk.GetEntity().GetType(), false)
	if err != nil {
		log.Error(err)
	}
	if !found {
		e.upsertRisk(risk)
		return risk
	}

	// replace old risk results with new ones.
	riskResultMap := make(map[string]*storage.Risk_Result)
	for _, result := range oldRisk.GetResults() {
		riskResultMap[result.GetName()] = result
	}
	for _, result := range risk.GetResults() {
		riskResultMap[result.GetName()] = result
	}

	riskResults := make([]*storage.Risk_Result, 0, len(riskResultMap))
	overallScore := float32(1.0)
	for _, result := range riskResultMap {
		overallScore *= result.GetScore()
		riskResults = append(riskResults, result)
	}
	risk.Results = riskResults
	risk.Score = overallScore
	e.upsertRisk(risk)

	return risk
}

func (e *managerImpl) upsertRisk(risk *storage.Risk) {
	if err := e.riskStorage.UpsertRisk(riskReprocessorCtx, risk); err != nil {
		log.Errorf("error reprocessing risk for %s %s: %v", risk.GetEntity().GetType().String(), risk.GetEntity().GetId(), err)
	}
}
