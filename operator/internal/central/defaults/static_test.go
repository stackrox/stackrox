package defaults

import (
	"testing"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/common/defaulting_test_helpers"
	"github.com/stretchr/testify/require"
)

func TestCentralStaticDefaults(t *testing.T) {
	tests := map[string]struct {
		defaults   *platform.CentralSpec
		errorCheck require.ErrorAssertionFunc
	}{
		"empty defaults": {
			defaults:   &platform.CentralSpec{},
			errorCheck: require.NoError,
		},
		"non-empty defaults": {
			defaults: &platform.CentralSpec{Egress: &platform.Egress{}},
			errorCheck: func(t require.TestingT, err error, i ...interface{}) {
				require.ErrorContains(t, err, "is not empty")
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tt.errorCheck(t, CentralStaticDefaults.DefaultingFunc(logr.Discard(), nil, nil, nil, tt.defaults))
		})
	}
}

func TestCentralStaticDefaultsMatchesCRD(t *testing.T) {
	centralSpecSchema := defaulting_test_helpers.LoadSpecSchema(t, "centrals")

	t.Run("Defaults", func(t *testing.T) {
		defaulting_test_helpers.CheckStruct(t, staticDefaults, centralSpecSchema)
	})
}
