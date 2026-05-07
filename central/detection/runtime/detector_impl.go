package runtime

import (
	"context"

	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection"
	"github.com/stackrox/rox/generated/storage"
	detectionPkg "github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	detectorCtx = sac.WithAllAccess(context.Background())
)

type detectorImpl struct {
	policySet   detection.PolicySet
	deployments datastore.DataStore
}

// PolicySet retrieves the policy set.
func (d *detectorImpl) PolicySet() detection.PolicySet {
	return d.policySet
}

func (d *detectorImpl) DeploymentWhitelistedForPolicy(deploymentID, policyID string) bool {
	var result bool
	err := d.policySet.ForOne(policyID, func(compiled detectionPkg.CompiledPolicy) error {
		if compiled.Policy().GetDisabled() {
			result = true
			return nil
		}
		dep, exists, err := d.deployments.GetDeployment(detectorCtx, deploymentID)
		if err != nil {
			return err
		}
		if !exists || (features.DeploymentSoftDeletion.Enabled() && dep.GetState() == storage.DeploymentState_DEPLOYMENT_STATE_DELETED) {
			// Assume it's not excluded if it doesn't exist or is soft-deleted,
			// otherwise runtime alerts for deleted deployments will always get
			// removed every time we update a policy.
			result = false
			return nil
		}
		result = !compiled.AppliesTo(detectorCtx, dep)
		return nil
	})
	if err != nil {
		log.Errorf("Couldn't evaluate exclusion for deployment %s, policy %s: %s", deploymentID, policyID, err)
	}
	return result
}

func (d *detectorImpl) DeploymentInactive(deploymentID string) bool {
	dep, exists, err := d.deployments.ListDeployment(detectorCtx, deploymentID)
	if err != nil {
		log.Errorf("Couldn't determine inactive state of deployment %q: %v", deploymentID, err)
		return false
	}
	return !exists || (features.DeploymentSoftDeletion.Enabled() && dep.GetState() == storage.DeploymentState_DEPLOYMENT_STATE_DELETED)
}
