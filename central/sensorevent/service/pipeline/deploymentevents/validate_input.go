package deploymentevents

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

func newValidateInput() *validateInputImpl {
	return &validateInputImpl{}
}

type validateInputImpl struct{}

func (s *validateInputImpl) do(deployment *v1.Deployment) error {
	// validate input.
	if deployment == nil {
		return fmt.Errorf("deployment must not be empty")
	}
	return nil
}
