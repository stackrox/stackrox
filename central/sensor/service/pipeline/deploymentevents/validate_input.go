package deploymentevents

import (
	"github.com/pkg/errors"

	"github.com/stackrox/stackrox/generated/storage"
)

func newValidateInput() *validateInputImpl {
	return &validateInputImpl{}
}

type validateInputImpl struct{}

func (s *validateInputImpl) do(deployment *storage.Deployment) error {
	// validate input.
	if deployment == nil {
		return errors.New("deployment must not be empty")
	}
	return nil
}
