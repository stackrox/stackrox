package connection

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/tests/helper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

var (
	NginxDeployment1 = helper.K8sResourceInfo{Kind: "Deployment", YamlFile: "nginx.yaml", Name: "nginx-deployment"}
)

func Test_SensorHello(t *testing.T) {
	t.Setenv("ROX_PREVENT_SENSOR_RESTART_ON_DISCONNECT", "true")
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_INITIAL_INTERVAL", "1s")
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL", "2s")

	c, err := helper.NewContextWithConfig(t, helper.CentralConfig{
		InitialSystemPolicies: nil,
		CertFilePath:          "../../../tools/local-sensor/certs/",
	})
	t.Cleanup(c.Stop)

	require.NoError(t, err)

	c.RunTest(t, helper.WithTestCase(func(t *testing.T, testContext *helper.TestContext, _ map[string]k8s.Object) {
		hello1 := testContext.WaitForHello(t, 3*time.Minute)
		require.NotNil(t, hello1)
		assert.Equal(t, central.SensorHello_STARTUP, hello1.GetSensorState())
		testContext.RestartFakeCentralConnection()
		hello2 := testContext.WaitForHello(t, 3*time.Minute)
		require.NotNil(t, hello2)
		assert.Equal(t, central.SensorHello_RECONNECT, hello2.GetSensorState())
	}))

}

func Test_SensorHello2(t *testing.T) {
	t.Setenv("ROX_PREVENT_SENSOR_RESTART_ON_DISCONNECT", "true")
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_INITIAL_INTERVAL", "1s")
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL", "2s")

	c, err := helper.NewContextWithConfig(t, helper.CentralConfig{
		InitialSystemPolicies: nil,
		CertFilePath:          "../../../tools/local-sensor/certs/",
	})
	t.Cleanup(c.Stop)

	require.NoError(t, err)

	c.RunTest(t, helper.WithTestCase(func(t *testing.T, testContext *helper.TestContext, _ map[string]k8s.Object) {
		hello1 := testContext.WaitForHello(t, 3*time.Minute)
		require.NotNil(t, hello1)
		assert.Equal(t, central.SensorHello_STARTUP, hello1.GetSensorState())
		testContext.RestartFakeCentralConnection()
		hello2 := testContext.WaitForHello(t, 3*time.Minute)
		require.NotNil(t, hello2)
		assert.Equal(t, central.SensorHello_RECONNECT, hello2.GetSensorState())
	}))
}

func Test_SensorReconnects(t *testing.T) {
	t.Setenv("ROX_PREVENT_SENSOR_RESTART_ON_DISCONNECT", "true")
	t.Setenv("ROX_RESYNC_DISABLED", "true")
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_INITIAL_INTERVAL", "1s")
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL", "2s")

	c, err := helper.NewContextWithConfig(t, helper.CentralConfig{
		InitialSystemPolicies: nil,
		CertFilePath:          "../../../tools/local-sensor/certs/",
	})

	t.Cleanup(c.Stop)

	require.NoError(t, err)

	c.RunTest(t, helper.WithTestCase(func(t *testing.T, testContext *helper.TestContext, _ map[string]k8s.Object) {

		// This test case will make sure that:
		//  1) Sensor does not crash when ROX_PREVENT_SENSOR_RESTART_ON_DISCONNECT is set.
		//  2) After Central reconnects, events can continue streaming messages
		//
		// Since this a base test, which will be used to test further features in the ROX-9776 epic, there are some
		// caveats. Because the receiving-end of component's `ResponseC` channel is a goroutine in `CentralSender`
		// implementation, any messages read by the channel after the gRPC connection stopped will be dropped. This is
		// because the entire instance of `CentralSender` is destroyed (i.e. pointer dereferences in favor of new instance)
		// and it's eventually cleaned up by GC.
		//
		// This can cause messages to be lost (e.g. NginxDeployment2 in this test) if they are received too close to gRPC shutdown.
		// For this reason we've added a sleep timer here. To make sure the event is received after the connection was
		// re-established.
		//
		// Note that this behavior is *not* acceptable in production. There are follow-up tasks (ROX-17327 and ROX-17157)
		// that will tackle this, and only then the sleep timer could be removed.

		testContext.WaitForSyncEvent(t, 2*time.Minute)

		// Stop fake central gRPC server and create a new one immediately after.
		testContext.RestartFakeCentralConnection()
		testContext.WaitForSyncEvent(t, 2*time.Minute)

		assert.False(t, testContext.SensorStopped())

		// We applied the resource _after_ Sensor restarted. Now we should check that this deployment will be sent to Central.
		_, err = c.ApplyResourceAndWaitNoObject(context.Background(), t, helper.DefaultNamespace, NginxDeployment1, nil)
		require.NoError(t, err)
	}))
}
