package policy

import (
	"fmt"

	"github.com/pkg/errors"
)

var (
	// ErrBreakingPolicies occurs if policies are found whose storage.EnforcementAction leads
	// to a failure (i.e. storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT when checking images)
	ErrBreakingPolicies = errors.New("failed policies found")
)

// NewErrBreakingPolicies creates a ErrBreakingPolicies with the number of policies within the explanation.
func NewErrBreakingPolicies(numOfBreakingPolicies int) error {
	return fmt.Errorf("%w: %d policies violated that are failing the check",
		ErrBreakingPolicies, numOfBreakingPolicies)
}
