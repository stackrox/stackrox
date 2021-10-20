package check

import (
	"fmt"

	"github.com/pkg/errors"
)

var (
	// errFailedPoliciesFound occurs if policies are found whose storage.EnforcementAction leads
	// to a failure (i.e. storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT when checking images)
	errFailedPoliciesFound = errors.New("failed policies found")
)

// newErrFailedPoliciesFound
func newErrFailedPoliciesFound(numOfFailedPolicies int) error {
	return fmt.Errorf("%w: %d policies violated that are failing the check",
		errFailedPoliciesFound, numOfFailedPolicies)
}
