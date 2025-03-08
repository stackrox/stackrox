package openshift

import (
	"strconv"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestGlobalPullSecret(t *testing.T) {
	tcs := []struct {
		ns       string
		name     string
		expected bool
	}{
		{GlobalPullSecretNamespace, GlobalPullSecretName, true},
		{GlobalPullSecretName, GlobalPullSecretNamespace, false},
		{GlobalPullSecretNamespace, "fake", false},
		{"fake", GlobalPullSecretName, false},
		{"fake", "fake", false},
	}
	for i, tc := range tcs {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tc.expected, GlobalPullSecret(tc.ns, tc.name))

			assert.Equal(t, tc.expected, GlobalPullSecretIntegration(&storage.ImageIntegration{
				Source: &storage.ImageIntegration_Source{
					Namespace:           tc.ns,
					ImagePullSecretName: tc.name,
				},
			}))

		})
	}

	t.Run("edge cases", func(t *testing.T) {
		assert.False(t, GlobalPullSecretIntegration(nil))
		assert.False(t, GlobalPullSecretIntegration(&storage.ImageIntegration{}))
		assert.False(t, GlobalPullSecretIntegration(&storage.ImageIntegration{Source: &storage.ImageIntegration_Source{}}))
	})
}
