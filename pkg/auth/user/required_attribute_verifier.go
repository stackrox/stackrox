package user

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sliceutils"
)

// NewRequiredAttributesVerifier returns an AttributeVerifier that will verify all
// attributes of a user and return an error if any of the required attributes are missing or
// have a value that is different than expected.
func NewRequiredAttributesVerifier(requiredAttributes []*storage.AuthProvider_RequiredAttribute) AttributeVerifier {
	return &checkRequiredAttributesImpl{
		required: requiredAttributes,
	}
}

type checkRequiredAttributesImpl struct {
	required []*storage.AuthProvider_RequiredAttribute
}

func (c checkRequiredAttributesImpl) Verify(attributes map[string][]string) error {
	// User attributes do not _specifically_ have to be set, handle this explicitly.
	if attributes == nil {
		return errox.NoCredentials.CausedBy("none of the required attributes set")
	}

	for _, required := range c.required {
		if observedValue, ok := attributes[required.GetAttributeKey()]; !ok || observedValue == nil {
			// Explicitly return 401, as we do not want clients to be issued a token.
			return errox.NoCredentials.CausedByf("missing required attribute %q", required.GetAttributeKey())
		} else if ok && sliceutils.StringFind(observedValue, required.GetAttributeValue()) == -1 {
			// Explicitly return 401, as we do not want clients to be issued a token.
			return errox.NoCredentials.CausedByf("required attribute %q did not have the required value",
				required.GetAttributeKey())
		}
	}
	return nil
}
