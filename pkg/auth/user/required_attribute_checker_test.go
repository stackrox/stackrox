package user

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckRequiredAttributesImpl_Check(t *testing.T) {
	cases := map[string]struct {
		shouldFail  bool
		expectedErr error
		attributes  []*storage.AuthProvider_RequiredAttribute
		userDesc    *permissions.UserDescriptor
	}{
		"required attribute set should not fail": {
			attributes: []*storage.AuthProvider_RequiredAttribute{
				{AttributeName: "required-attribute", AttributeValue: "some-value"},
			},
			userDesc: &permissions.UserDescriptor{
				Attributes: map[string][]string{"required-attribute": {"some-value"}},
			},
		},
		"required attribute not set should fail": {
			attributes: []*storage.AuthProvider_RequiredAttribute{
				{AttributeName: "required-attribute", AttributeValue: "some-value"},
			},
			userDesc: &permissions.UserDescriptor{
				Attributes: map[string][]string{"other-attribute": {"some-value"}},
			},
			expectedErr: errox.NoCredentials,
			shouldFail:  true,
		},
		"no attribute set should fail": {
			attributes: []*storage.AuthProvider_RequiredAttribute{
				{AttributeName: "required-attribute", AttributeValue: "some-value"},
			},
			userDesc: &permissions.UserDescriptor{
				Attributes: nil,
			},
			expectedErr: errox.NoCredentials,
			shouldFail:  true,
		},
		"multiple required attributes set should not fail": {
			attributes: []*storage.AuthProvider_RequiredAttribute{
				{AttributeName: "required-attribute", AttributeValue: "some-value"},
				{AttributeName: "another-required-attribute", AttributeValue: "another-value"},
			},
			userDesc: &permissions.UserDescriptor{
				Attributes: map[string][]string{
					"required-attribute":         {"some-value"},
					"another-required-attribute": {"another-value"},
				},
			},
		},
		"only some required attributes set should fail": {
			attributes: []*storage.AuthProvider_RequiredAttribute{
				{AttributeName: "required-attribute", AttributeValue: "some-value"},
			},
			userDesc: &permissions.UserDescriptor{
				Attributes: map[string][]string{
					"another-required-attribute": {"another-value"},
				},
			},
			expectedErr: errox.NoCredentials,
			shouldFail:  true,
		},
		"required attribute in map but nil value should fail": {
			attributes: []*storage.AuthProvider_RequiredAttribute{
				{AttributeName: "required-attribute", AttributeValue: "some-value"},
			},
			userDesc: &permissions.UserDescriptor{
				Attributes: map[string][]string{
					"required-attribute": nil,
				},
			},
			expectedErr: errox.NoCredentials,
			shouldFail:  true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			checker := NewRequiredAttributesChecker(c.attributes)
			err := checker.Check(c.userDesc)
			if c.shouldFail {
				require.Error(t, err)
				assert.ErrorIs(t, err, c.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
