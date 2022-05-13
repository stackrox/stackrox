package user

import (
	"strings"

	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
)

// NewRequiredAttributesChecker returns an AttributeChecker that will verify all
// attributes of a user and return an error if any of the required attributes are missing.
func NewRequiredAttributesChecker(requiredAttributes ...string) AttributeChecker {
	return &checkRequiredAttributesImpl{
		attributes: requiredAttributes,
	}
}

type checkRequiredAttributesImpl struct {
	attributes []string
}

func (c checkRequiredAttributesImpl) Check(userDescriptor *permissions.UserDescriptor) error {
	// User attributes do not _specifically_ have to be set, handle this explicitly.
	if userDescriptor.Attributes == nil {
		return errox.NoCredentials.CausedByf("none of the required attributes [%s] set",
			strings.Join(c.attributes, ", "))
	}

	for _, attribute := range c.attributes {
		if val, ok := userDescriptor.Attributes[attribute]; !ok || val == nil {
			// Explicitly return 403, as we do not want clients to be issued a token.
			return errox.NoCredentials.CausedByf("missing required attribute %q", attribute)
		}
	}
	return nil
}
