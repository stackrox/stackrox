package manager

import (
	"context"
	"time"

	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/risk"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
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
}

type managerImpl struct {
	deploymentStorage deploymentDS.DataStore
	riskStorage       riskDS.DataStore
	scorer            risk.Scorer
}

// New returns a new manager
func New(deploymentStorage deploymentDS.DataStore, riskStorage riskDS.DataStore,
	scorer risk.Scorer) (Manager, error) {
	m := &managerImpl{
		deploymentStorage: deploymentStorage,
		riskStorage:       riskStorage,
		scorer:            scorer,
	}
	return m, nil
}

// ReprocessDeploymentRisk will reprocess the passed deployments risk and save the results
func (e *managerImpl) ReprocessDeploymentRiskWithImages(deployment *storage.Deployment, images []*storage.Image) {
	defer metrics.ObserveRiskProcessingDuration(time.Now())

	risk := e.scorer.Score(allAccessCtx, deployment, images)
	if err := e.riskStorage.UpsertRisk(riskReprocessorCtx, risk); err != nil {
		log.Errorf("Error reprocessing risk for deployment %s: %s", deployment.GetName(), err)
	}
}

// ReprocessDeploymentRisk will reprocess the passed deployments risk and save the results
func (e *managerImpl) ReprocessDeploymentRisk(deployment *storage.Deployment) {
	images, err := e.deploymentStorage.GetImagesForDeployment(riskReprocessorCtx, deployment)
	if err != nil {
		log.Errorf("error fetching images for deployment %s: %v", deployment.GetName(), err)
		return
	}
	e.ReprocessDeploymentRiskWithImages(deployment, images)
}
