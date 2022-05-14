package user

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sliceutils"
)

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
		return errox.NoCredentials.CausedByf("none of the required attributes [%s] set",
			formatRequiredAttributes(c.attributes))
	}

	for _, attribute := range c.attributes {
		if userAttributes, ok := userDescriptor.Attributes[attribute.AttributeName]; !ok || userAttributes == nil ||
			sliceutils.StringFind(userAttributes, attribute.AttributeValue) == -1 {
			// Explicitly return 403, as we do not want clients to be issued a token.
			return errox.NoCredentials.CausedByf("missing required attribute %s=%s", attribute.AttributeName,
				attribute.AttributeValue)
		}
	}
	return nil
}

func formatRequiredAttributes(attributes []*storage.AuthProvider_RequiredAttribute) string {
	attrKeyValue := make([]string, 0, len(attributes))
	for _, attr := range attributes {
		attrKeyValue = append(attrKeyValue, fmt.Sprintf("%s=%s", attr.AttributeName, attr.AttributeValue))
	}
	return strings.Join(attrKeyValue, ",")
}
