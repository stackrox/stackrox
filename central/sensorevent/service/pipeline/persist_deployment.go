package pipeline

import (
	"bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

func newPersistDeployment(deployments datastore.DataStore) *persistDeploymentImpl {
	return &persistDeploymentImpl{
		deployments: deployments,
	}
}

type persistDeploymentImpl struct {
	deployments datastore.DataStore
}

func (s *persistDeploymentImpl) do(event *v1.DeploymentEvent) error {
	action := event.GetAction()
	deployment := event.GetDeployment()
	switch action {
	case v1.ResourceAction_PREEXISTING_RESOURCE, v1.ResourceAction_CREATE_RESOURCE:
		if err := s.deployments.UpdateDeployment(deployment); err != nil {
			log.Errorf("unable to add deployment %s: %s", deployment.GetId(), err)
			return err
		}
	case v1.ResourceAction_UPDATE_RESOURCE:
		if err := s.deployments.UpdateDeployment(deployment); err != nil {
			log.Errorf("unable to update deployment %s: %s", deployment.GetId(), err)
			return err
		}
	case v1.ResourceAction_REMOVE_RESOURCE:
		if err := s.deployments.RemoveDeployment(deployment.GetId()); err != nil {
			log.Errorf("unable to remove deployment %s: %s", deployment.GetId(), err)
			return err
		}
	default:
		log.Warnf("unknown action: %s", action)
		return nil // Be interoperable: don't reject these requests.
	}
	return nil
}
