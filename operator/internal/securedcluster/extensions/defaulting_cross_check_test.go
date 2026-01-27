package extensions

import (
	"testing"

	"github.com/go-logr/logr/testr"
	"github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/common/defaulting_test_helpers"
	"github.com/stretchr/testify/require"
)

func TestSecuredClusterDefaultsMatchCRD(t *testing.T) {
	SecuredClusterSpecSchema := defaulting_test_helpers.LoadSpecSchema(t, "securedclusters")
	sc := &v1alpha1.SecuredCluster{}
	for _, flow := range defaultingFlows {
		require.NoError(t, flow.DefaultingFunc(testr.New(t), &v1alpha1.SecuredClusterStatus{}, map[string]string{}, &sc.Spec, &sc.Defaults))
	}

	t.Run("Defaults", func(t *testing.T) {
		defaulting_test_helpers.CheckStruct(t, sc.Defaults, SecuredClusterSpecSchema)
	})
}
