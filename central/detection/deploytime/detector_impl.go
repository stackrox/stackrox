package deploytime

import (
	"context"

	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	log = logging.LoggerForModule()

	executorCtx = sac.WithAllAccess(context.Background())
)

type detectorImpl struct {
	policySet   detection.PolicySet
	deployments deploymentDataStore.DataStore
}

// UpsertPolicy adds or updates a policy in the set.
func (d *detectorImpl) PolicySet() detection.PolicySet {
	return d.policySet
}

// Detect runs detection on an deployment, returning any generated alerts.
func (d *detectorImpl) Detect(ctx DetectionContext, deployment *storage.Deployment, images []*storage.Image) ([]*storage.Alert, error) {
	exe := newSingleDeploymentExecutor(executorCtx, ctx, d.deployments, deployment, images)
	err := d.policySet.ForEach(exe)
	if err != nil {
		return nil, err
	}
	return exe.GetAlerts(), nil
}

func (d *detectorImpl) AlertsForPolicy(policyID string) ([]*storage.Alert, error) {
	exe := newAllDeploymentsExecutor(executorCtx, d.deployments)
	err := d.policySet.ForOne(policyID, exe)
	if err != nil {
		return nil, err
	}
	return exe.GetAlerts(), nil
}
