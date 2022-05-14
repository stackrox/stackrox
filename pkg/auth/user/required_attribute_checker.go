package user

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sliceutils"
)

var log = logging.LoggerForModule()

// NewRequiredAttributesChecker returns an AttributeChecker that will verify all
// attributes of a user and return an error if any of the required attributes are missing.
func NewRequiredAttributesChecker(requiredAttributes []*storage.AuthProvider_RequiredAttribute) AttributeChecker {
	return &checkRequiredAttributesImpl{
		attributes: requiredAttributes,
	}
}

type checkRequiredAttributesImpl struct {
	attributes []*storage.AuthProvider_RequiredAttribute
}

func (c checkRequiredAttributesImpl) Check(userDescriptor *permissions.UserDescriptor) error {
	// User attributes do not _specifically_ have to be set, handle this explicitly.
	if userDescriptor.Attributes == nil {
		return errox.NoCredentials.CausedBy("none of the required attributes set")
	}

	for _, attribute := range c.attributes {
		if userAttributes, ok := userDescriptor.Attributes[attribute.AttributeKey]; !ok || userAttributes == nil ||
			sliceutils.StringFind(userAttributes, attribute.AttributeValue) == -1 {
			log.Infof("Missing attribute %q; available attributes: %q", attribute.AttributeKey, strings.Join(userAttributes, ", "))
			// Explicitly return 401, as we do not want clients to be issued a token.
			return errox.NoCredentials.CausedBy("missing required attribute(s)")
		}
	}
	return nil
}
