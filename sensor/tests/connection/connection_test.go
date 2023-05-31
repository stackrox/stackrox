package connection

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/tests/helper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

var (
	NginxDeployment = helper.K8sResourceInfo{Kind: "Deployment", YamlFile: "nginx.yaml"}
)

func Test_ConnectionBroke(t *testing.T) {
	t.Skip()
	t.Setenv("ROX_PREVENT_SENSOR_RESTART_ON_DISCONNECT", "true")
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_INITIAL_INTERVAL", "1s")
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL", "2s")

	c, err := helper.NewContext(t)
	require.NoError(t, err)

	c.RunTest(helper.WithTestCase(func(t *testing.T, testContext *helper.TestContext, _ map[string]k8s.Object) {
		fakeCentral := testContext.GetFakeCentral()

		// TODO: Create a signal that can be used to notify that Sensor is ready
		t.Logf("Waiting 5s until sensor starts")
		time.Sleep(5 * time.Second)

		// Force gRPC server to stop
		fakeCentral.ServerPointer.Stop()

		// Give it double the retry interval to make sure sensor started retrying
		time.Sleep(2 * time.Second)

		assert.False(t, testContext.SensorStopped())
	}))
}

func Test_SensorReconnects(t *testing.T) {
	t.Setenv("ROX_PREVENT_SENSOR_RESTART_ON_DISCONNECT", "true")
	//t.Setenv("ROX_SENSOR_CONNECTION_RETRY_INITIAL_INTERVAL", "1s")
	//t.Setenv("ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL", "2s")

	c, err := helper.NewContext(t)
	require.NoError(t, err)

	c.RunTest(helper.WithTestCase(func(t *testing.T, testContext *helper.TestContext, _ map[string]k8s.Object) {
		// TODO: Create a signal that can be used to notify that Sensor is ready
		t.Logf("Waiting 5s until sensor starts")
		time.Sleep(5 * time.Second)

		testContext.NewFakeCentralConnection()

		// Give it double the retry interval to make sure sensor started retrying
		time.Sleep(2 * time.Second)

		assert.False(t, testContext.SensorStopped())

		deleteDeployment, err := testContext.ApplyResourceNoObject(context.Background(), helper.DefaultNamespace, NginxDeployment, nil)
		defer utils.IgnoreError(deleteDeployment)

		require.NoError(t, err)

		// We applied the resource _after_ Sensor restarted. Now we should check that this deployment will be sent to Central.
		testContext.DeploymentCreateReceived("nginx-deployment")
	}))
}
