package policy

import (
	"fmt"

	"github.com/pkg/errors"
)

var (
	// ErrFailedPolicies occurs if policies are found whose storage.EnforcementAction leads
	// to a failure (i.e. storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT when checking images)
	ErrFailedPolicies = errors.New("failed policies found")
)

// NewErrFailedPolicies creates a ErrFailedPolicies with the number of failed policies within the explanation.
func NewErrFailedPolicies(numOfFailedPolicies int) error {
	return fmt.Errorf("%w: %d policies violated that are failing the check",
		ErrFailedPolicies, numOfFailedPolicies)
}
