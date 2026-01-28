package extensions

import (
	"testing"

	"github.com/go-logr/logr/testr"
	"github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/common/defaulting_test_helpers"
	"github.com/stretchr/testify/require"
)

func TestCentralDefaultsMatchCRD(t *testing.T) {
	centralSpecSchema := defaulting_test_helpers.LoadSpecSchema(t, "centrals")
	c := &v1alpha1.Central{}
	for _, flow := range defaultingFlows {
		require.NoError(t, flow.DefaultingFunc(testr.New(t), &v1alpha1.CentralStatus{}, map[string]string{}, &c.Spec, &c.Defaults))
	}
	t.Run("Defaults", func(t *testing.T) {
		defaulting_test_helpers.CheckStruct(t, c.Defaults, centralSpecSchema)
	})
}
