package defaults

import (
	"testing"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/common/defaulting_test_helpers"
	"github.com/stretchr/testify/require"
)

func TestSecuredClusterStaticDefaults(t *testing.T) {
	tests := map[string]struct {
		defaults   *platform.SecuredClusterSpec
		errorCheck require.ErrorAssertionFunc
	}{
		"empty defaults": {
			defaults:   &platform.SecuredClusterSpec{},
			errorCheck: require.NoError,
		},
		"non-empty defaults": {
			defaults: &platform.SecuredClusterSpec{Customize: &platform.CustomizeSpec{}},
			errorCheck: func(t require.TestingT, err error, i ...interface{}) {
				require.ErrorContains(t, err, "is not empty")
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tt.errorCheck(t, SecuredClusterStaticDefaults.DefaultingFunc(logr.Discard(), nil, nil, nil, tt.defaults))
		})
	}
}

func TestSecuredClusterStaticDefaultsMatchesCRD(t *testing.T) {
	t.Setenv("ROX_ADMISSION_CONTROLLER_CONFIG", "true")
	SecuredClusterSpecSchema := defaulting_test_helpers.LoadSpecSchema(t, "securedclusters")

	t.Run("Defaults", func(t *testing.T) {
		defaulting_test_helpers.CheckStruct(t, staticDefaults, SecuredClusterSpecSchema)
	})
}
