package user

import "github.com/stackrox/rox/pkg/auth/permissions"

// AttributeChecker adds checks for attributes that must be present.
type AttributeChecker interface {
	// Check will verify the attributes and return an error
	// if specific checks are failing.
	Check(userDescriptor *permissions.UserDescriptor) error
}
