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

func (s *createResponseImpl) do(deployment *v1.Deployment, action v1.ResourceAction) *v1.SensorEventResponse {
	var alertID string
	var enforcement v1.EnforcementAction
	if action == v1.ResourceAction_REMOVE_RESOURCE {
		_ = s.onRemove(deployment)
	} else if action == v1.ResourceAction_CREATE_RESOURCE {
		// We only want enforcement if the deployment was just created.
		alertID, enforcement, _ = s.onUpdate(deployment)
	} else {
		alertID, _, _ = s.onUpdate(deployment)
	}

	if enforcement != v1.EnforcementAction_UNSET_ENFORCEMENT {
		log.Warnf("Taking enforcement action %s against deployment %s", enforcement, deployment.GetName())
	}

	response := new(v1.DeploymentEventResponse)
	response.DeploymentId = deployment.GetId()
	response.AlertId = alertID
	response.Enforcement = enforcement

	return &v1.SensorEventResponse{
		Resource: &v1.SensorEventResponse_Deployment{
			Deployment: response,
		},
	}
}
