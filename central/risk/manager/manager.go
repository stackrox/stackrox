package manager

import (
	"context"
	"time"

	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/risk"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	log            = logging.LoggerForModule()
	depAndImageCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment, resources.Image),
		))
	// Used for scorer.score() as the different Multipliers which will eventually use this context will require different permissions
	allAccessCtx = sac.WithAllAccess(context.Background())
)

// Manager manages changes to the risk of the deployments
type Manager interface {
	ReprocessDeploymentRisk(deployment *storage.Deployment)
}

type managerImpl struct {
	deploymentStorage deploymentDS.DataStore

	scorer risk.Scorer
}

// New returns a new manager
func New(deploymentStorage deploymentDS.DataStore,
	scorer risk.Scorer) (Manager, error) {
	m := &managerImpl{
		deploymentStorage: deploymentStorage,
		scorer:            scorer,
	}
	return m, nil
}

// ReprocessDeploymentRisk will reprocess the passed deployments risk and save the results
func (e *managerImpl) ReprocessDeploymentRisk(deployment *storage.Deployment) {
	deployment = protoutils.CloneStorageDeployment(deployment)
	if err := e.addRiskToDeployment(deployment); err != nil {
		log.Errorf("Error reprocessing risk for deployment %s: %s", deployment.GetName(), err)
	}
}

// addRiskToDeployment will add the risk
func (e *managerImpl) addRiskToDeployment(deployment *storage.Deployment) error {
	defer metrics.ObserveRiskProcessingDuration(time.Now())

	images, err := e.deploymentStorage.GetImagesForDeployment(depAndImageCtx, deployment)
	if err != nil {
		return err
	}

	deployment.Risk = e.scorer.Score(allAccessCtx, deployment, images)
	return e.deploymentStorage.UpdateDeployment(depAndImageCtx, deployment)
}
