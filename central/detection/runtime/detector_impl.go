package runtime

import (
	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection"
	"github.com/stackrox/rox/generated/storage"
)

type detectorImpl struct {
	policySet   detection.PolicySet
	deployments datastore.DataStore
}

// UpsertPolicy adds or updates a policy in the set.
func (d *detectorImpl) PolicySet() detection.PolicySet {
	return d.policySet
}

func (d *detectorImpl) AlertsForDeployments(deploymentIDs ...string) ([]*storage.Alert, error) {
	executor := newAlertCollectingExecutor(d.deployments, deploymentIDs...)
	err := d.policySet.ForEach(executor)
	if err != nil {
		return nil, err
	}

	return executor.GetAlerts(), nil
}

func (d *detectorImpl) AlertsForPolicy(policyID string) ([]*storage.Alert, error) {
	executor := newAlertCollectingExecutor(d.deployments)
	err := d.policySet.ForOne(policyID, executor)
	if err != nil {
		return nil, err
	}

	return executor.GetAlerts(), nil
}

func (d *detectorImpl) DeploymentWhitelistedForPolicy(deploymentID, policyID string) bool {
	executor := newWhitelistTestingExecutor(d.deployments, deploymentID)
	err := d.policySet.ForOne(policyID, executor)
	if err != nil {
		log.Errorf("Couldn't evaluate whitelist for deployment %s, policy %s: %s", deploymentID, policyID, err)
	}
	return executor.GetResult()
}
