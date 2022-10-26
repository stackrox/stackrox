package resources

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/message"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/rbac"
)

type deploymentDependencyResolution struct {
	store        *DeploymentStore
	serviceStore *ServiceStore
	rbac         rbac.Store
}

func NewDeploymentResolver(deploymentStore *DeploymentStore, serviceStore *ServiceStore, rbacStore rbac.Store) *deploymentDependencyResolution {
	return &deploymentDependencyResolution{
		store:        deploymentStore,
		serviceStore: serviceStore,
		rbac:         rbacStore,
	}
}

func (r *deploymentDependencyResolution) ProcessDependencies(ref message.DeploymentRef) (*storage.Deployment, error) {
	ids := set.NewStringSet(ref.Id)
	deploymentWraps := r.store.getDeploymentsByIDs(ref.Namespace, ids)
	if len(deploymentWraps) > 1 {
		return nil, errors.Errorf("should have single deployment with id %s in store but instead found %d", ref.Id, len(deploymentWraps))
	}

	if len(deploymentWraps) == 0 {
		// This probably means that the deployment was deleted before a dependent
		// was scheduled to be processed. This can be dropped.
		log.Debugf("Deployment id %s is not on local store. Must have been deleted", ref.Id)
		return nil, nil
	}

	// This is the prototype of a "snapshot"
	deploymentWrap := deploymentWraps[0].Clone()

	deploymentWrap.updatePortExposureFromStore(r.serviceStore)
	deploymentWrap.updateServiceAccountPermissionLevel(r.rbac.GetPermissionLevelForDeployment(deploymentWrap.GetDeployment()))
	if err := deploymentWrap.updateHash(); err != nil {
		return nil, errors.Errorf("UNEXPECTED: could not calculate hash of deployment %s: %v", deploymentWrap.GetId(), err)
	}

	// NOTE: We don't want to upsert the updated deployment in the local store
	// if running on the new pipeline, the relationship data is always computed
	// through this function, regardless of the event that originated.

	return deploymentWrap.GetDeployment(), nil
}
