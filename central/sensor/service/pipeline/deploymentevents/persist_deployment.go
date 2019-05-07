package deploymentevents

import (
	"context"

	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

func newPersistDeployment(deployments datastore.DataStore) *persistDeploymentImpl {
	return &persistDeploymentImpl{
		deployments: deployments,
	}
}

type persistDeploymentImpl struct {
	deployments datastore.DataStore
}

func (s *persistDeploymentImpl) do(action central.ResourceAction, deployment *storage.Deployment) error {
	ctx := context.TODO()

	switch action {
	case central.ResourceAction_CREATE_RESOURCE:
		if err := s.deployments.UpsertDeployment(ctx, deployment); err != nil {
			log.Errorf("unable to add deployment %s: %s", deployment.GetId(), err)
			return err
		}
	case central.ResourceAction_UPDATE_RESOURCE:
		if err := s.deployments.UpsertDeployment(ctx, deployment); err != nil {
			log.Errorf("unable to update deployment %s: %s", deployment.GetId(), err)
			return err
		}
	case central.ResourceAction_REMOVE_RESOURCE:
		if err := s.deployments.RemoveDeployment(ctx, deployment.GetClusterId(), deployment.GetId()); err != nil {
			log.Errorf("unable to remove deployment %s: %s", deployment.GetId(), err)
			return err
		}
	default:
		log.Warnf("unknown action: %s", action)
		return nil // Be interoperable: don't reject these requests.
	}
	return nil
}
