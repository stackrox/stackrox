package pipeline

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

func newCreateResponse(toEnforcement func(deployment *v1.Deployment, action v1.ResourceAction) (alertID string, enforcement v1.EnforcementAction)) *createResponseImpl {
	return &createResponseImpl{
		toEnforcement: toEnforcement,
	}
}

type createResponseImpl struct {
	toEnforcement func(deployment *v1.Deployment, action v1.ResourceAction) (alertID string, enforcement v1.EnforcementAction)
}

func (s *createResponseImpl) do(event *v1.DeploymentEvent) *v1.DeploymentEventResponse {
	alertID, enforcement := s.toEnforcement(event.GetDeployment(), event.GetAction())
	if enforcement != v1.EnforcementAction_UNSET_ENFORCEMENT {
		log.Warnf("Taking enforcement action %s against deployment %s", enforcement, event.GetDeployment().GetName())
	}

	response := new(v1.DeploymentEventResponse)
	response.DeploymentId = event.GetDeployment().GetId()
	response.AlertId = alertID
	response.Enforcement = enforcement
	return response
}
