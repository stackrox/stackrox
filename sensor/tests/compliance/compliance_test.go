package compliance

import (
	"testing"

	"github.com/stackrox/rox/sensor/tests/helper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

func Test_SensorCompliance(t *testing.T) {
	t.Setenv("ROX_PREVENT_SENSOR_RESTART_ON_DISCONNECT", "true")
	t.Setenv("ROX_RESYNC_DISABLED", "true")
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_INITIAL_INTERVAL", "1s")
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL", "2s")

	c, err := helper.NewContextWithConfig(t, helper.CentralConfig{
		InitialSystemPolicies: nil,
		CertFilePath:          "../../../tools/local-sensor/certs/",
	})

	require.NoError(t, err)

	c.RunTest(helper.WithTestCase(func(t *testing.T, testContext *helper.TestContext, _ map[string]k8s.Object) {
		assert.True(t, true)
	}))
}
