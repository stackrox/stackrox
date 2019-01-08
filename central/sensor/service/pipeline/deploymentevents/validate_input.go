package deploymentevents

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
)

func newValidateInput() *validateInputImpl {
	return &validateInputImpl{}
}

type validateInputImpl struct{}

func (s *validateInputImpl) do(deployment *storage.Deployment) error {
	// validate input.
	if deployment == nil {
		return fmt.Errorf("deployment must not be empty")
	}
	return nil
}
