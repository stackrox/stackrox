package user

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestCheckRequiredAttributesImpl_Check(t *testing.T) {
	cases := map[string]struct {
		shouldFail bool
		required   []*storage.AuthProvider_RequiredAttribute
		attributes map[string][]string
	}{
		"required attribute set should not fail": {
			required: []*storage.AuthProvider_RequiredAttribute{
				{AttributeKey: "required-attribute", AttributeValue: "some-value"},
			},
			attributes: map[string][]string{"required-attribute": {"some-value"}},
		},
		"required attribute not set should fail": {
			required: []*storage.AuthProvider_RequiredAttribute{
				{AttributeKey: "required-attribute", AttributeValue: "some-value"},
			},
			attributes: map[string][]string{"other-attribute": {"some-value"}},
			shouldFail: true,
		},
		"no attribute set should fail": {
			required: []*storage.AuthProvider_RequiredAttribute{
				{AttributeKey: "required-attribute", AttributeValue: "some-value"},
			},
			attributes: nil,
			shouldFail: true,
		},
		"multiple required attributes set should not fail": {
			required: []*storage.AuthProvider_RequiredAttribute{
				{AttributeKey: "required-attribute", AttributeValue: "some-value"},
				{AttributeKey: "another-required-attribute", AttributeValue: "another-value"},
			},
			attributes: map[string][]string{
				"required-attribute":         {"some-value"},
				"another-required-attribute": {"another-value"},
			},
		},
		"only some required attributes set should fail": {
			required: []*storage.AuthProvider_RequiredAttribute{
				{AttributeKey: "required-attribute", AttributeValue: "some-value"},
				{AttributeKey: "another-required-attribute", AttributeValue: "another-value"},
			},
			attributes: map[string][]string{
				"another-required-attribute": {"another-value"},
			},
			shouldFail: true,
		},
		"required attribute in map but nil value should fail": {
			required: []*storage.AuthProvider_RequiredAttribute{
				{AttributeKey: "required-attribute", AttributeValue: "some-value"},
			},
			attributes: map[string][]string{
				"required-attribute": nil,
			},
			shouldFail: true,
		},
		"required attribute set but value does not match": {
			required: []*storage.AuthProvider_RequiredAttribute{
				{AttributeKey: "required-attribute", AttributeValue: "some-value"},
			},
			attributes: map[string][]string{"required-attribute": {"other-value"}},
			shouldFail: true,
		},
		"required attribute in map but empty array value should fail": {
			required: []*storage.AuthProvider_RequiredAttribute{
				{AttributeKey: "required-attribute", AttributeValue: "some-value"},
			},
			attributes: map[string][]string{"required-attribute": {}},
			shouldFail: true,
		},
		"required attribute in map but empty string value should fail": {
			required: []*storage.AuthProvider_RequiredAttribute{
				{AttributeKey: "required-attribute", AttributeValue: "some-value"},
			},
			attributes: map[string][]string{"required-attribute": {""}},
			shouldFail: true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			verifier := NewRequiredAttributesVerifier(c.required)
			err := verifier.Verify(c.attributes)
			if c.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
