package deploymentevents

import (
	"github.com/stackrox/rox/generated/api/v1"
)

func newCreateResponse(onUpdate func(deployment *v1.Deployment) (string, v1.EnforcementAction, error),
	onRemove func(deployment *v1.Deployment) error) *createResponseImpl {
	return &createResponseImpl{
		onUpdate: onUpdate,
		onRemove: onRemove,
	}
}

type createResponseImpl struct {
	onUpdate func(deployment *v1.Deployment) (string, v1.EnforcementAction, error)
	onRemove func(deployment *v1.Deployment) error
}

func (s *createResponseImpl) do(deployment *v1.Deployment, action v1.ResourceAction) *v1.SensorEnforcement {
	var alertID string
	var enforcement v1.EnforcementAction
	var err error
	if action == v1.ResourceAction_REMOVE_RESOURCE {
		err = s.onRemove(deployment)
	} else if action == v1.ResourceAction_CREATE_RESOURCE {
		// We only want enforcement if the deployment was just created.
		alertID, enforcement, err = s.onUpdate(deployment)
	} else {
		_, _, err = s.onUpdate(deployment)
	}
	if err != nil {
		log.Errorf("updating from deployment failed: %s", err)
	}

	if enforcement == v1.EnforcementAction_UNSET_ENFORCEMENT {
		return nil
	}

	// Only form and return the response if there is an enforcement action to be taken.
	response := new(v1.DeploymentEnforcement)
	response.DeploymentId = deployment.GetId()
	response.DeploymentName = deployment.GetName()
	response.DeploymentType = deployment.GetType()
	response.Namespace = deployment.GetNamespace()
	response.AlertId = alertID

	return &v1.SensorEnforcement{
		Enforcement: enforcement,
		Resource: &v1.SensorEnforcement_Deployment{
			Deployment: response,
		},
	}
}
