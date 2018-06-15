package pipeline

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

func newValidateInput() *validateInputImpl {
	return &validateInputImpl{}
}

type validateInputImpl struct{}

func (s *validateInputImpl) do(event *v1.DeploymentEvent) error {
	// validate input.
	if event == nil {
		return fmt.Errorf("event must not be empty")
	}
	if event.GetDeployment() == nil {
		return fmt.Errorf("event must include a deployment")
	}
	return nil
}
