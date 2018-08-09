package deploymentevents

import (
	"github.com/stackrox/rox/generated/api/v1"
)

func newCreateResponse(toEnforcement func(deployment *v1.Deployment, action v1.ResourceAction) (alertID string, enforcement v1.EnforcementAction)) *createResponseImpl {
	return &createResponseImpl{
		toEnforcement: toEnforcement,
	}
}

type createResponseImpl struct {
	toEnforcement func(deployment *v1.Deployment, action v1.ResourceAction) (alertID string, enforcement v1.EnforcementAction)
}

func (s *createResponseImpl) do(action v1.ResourceAction, deployment *v1.Deployment) *v1.SensorEventResponse {
	alertID, enforcement := s.toEnforcement(deployment, action)
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
