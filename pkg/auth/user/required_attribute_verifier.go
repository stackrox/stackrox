package user

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sliceutils"
)

// NewRequiredAttributesVerifier returns an AttributeVerifier that will verify all
// attributes and return an error if any of the required attributes are missing or
// have a value that is different from expected.
func NewRequiredAttributesVerifier(requiredAttributes []*storage.AuthProvider_RequiredAttribute) AttributeVerifier {
	return &checkRequiredAttributesImpl{
		required: requiredAttributes,
	}
}

type checkRequiredAttributesImpl struct {
	required []*storage.AuthProvider_RequiredAttribute
}

func (c *checkRequiredAttributesImpl) Verify(attributes map[string][]string) error {
	// Attributes could be empty, handle this specifically.
	if attributes == nil {
		return fmt.Errorf("none of the required attributes [%s] are set", attributeKeysAsString(c.required))
	}

	var missingAttributes []string
	for _, required := range c.required {
		if observedValue, ok := attributes[required.GetAttributeKey()]; !ok || observedValue == nil {
			missingAttributes = append(missingAttributes, required.GetAttributeKey())
		} else if ok && sliceutils.Find(observedValue, required.GetAttributeValue()) == -1 {
			return fmt.Errorf("required attribute %q did not have the required value", required.GetAttributeKey())
		}
	}

	if len(missingAttributes) > 0 {
		return fmt.Errorf("missing required attributes [%s]", strings.Join(missingAttributes, ","))
	}

	return nil
}

func attributeKeysAsString(attributes []*storage.AuthProvider_RequiredAttribute) string {
	keys := make([]string, 0, len(attributes))
	for _, attr := range attributes {
		keys = append(keys, attr.GetAttributeKey())
	}
	return strings.Join(keys, ",")
}
