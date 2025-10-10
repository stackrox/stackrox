package connection

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/tests/helper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

var (
	NginxDeployment1 = helper.K8sResourceInfo{Kind: "Deployment", YamlFile: "nginx.yaml", Name: "nginx-deployment"}
)

func Test_SensorHello(t *testing.T) {
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_INITIAL_INTERVAL", "1s")
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL", "2s")

	c, err := helper.NewContextWithConfig(t, helper.Config{
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
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_INITIAL_INTERVAL", "1s")
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL", "2s")

	c, err := helper.NewContextWithConfig(t, helper.Config{
		InitialSystemPolicies: nil,
		CertFilePath:          "../../../tools/local-sensor/certs/",
	})

	t.Cleanup(c.Stop)

	require.NoError(t, err)

	c.RunTest(t, helper.WithTestCase(func(t *testing.T, testContext *helper.TestContext, _ map[string]k8s.Object) {

		// This test case will make sure that:
		//  1) Sensor does not crash when connection to Central is interrupted.
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

func Test_KernelObjectProxy(t *testing.T) {
	c, err := helper.NewContextWithConfig(t, helper.Config{
		InitialSystemPolicies: nil,
		CertFilePath:          "../../../tools/local-sensor/certs/",
	})

	t.Cleanup(c.Stop)

	require.NoError(t, err)

	c.RunTest(t, helper.WithTestCase(func(t *testing.T, testContext *helper.TestContext, _ map[string]k8s.Object) {
		client := helper.NewHTTPTestClient(t, storage.ServiceType_COLLECTOR_SERVICE)
		req, err := http.NewRequest(http.MethodGet, "https://localhost:8443/kernel-objects/2.9.0/collector-ebpf-6.5.0-15-generic.o.gz", nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, "response: %v", resp)
	}))
}

func Test_ScannerDefinitionsProxy(t *testing.T) {
	t.Setenv("ROX_LOCAL_IMAGE_SCANNING_ENABLED", "true")
	c, err := helper.NewContextWithConfig(t, helper.Config{
		InitialSystemPolicies: nil,
		CertFilePath:          "../../../tools/local-sensor/certs/",
	})

	t.Cleanup(c.Stop)

	require.NoError(t, err)

	c.RunTest(t, helper.WithTestCase(func(t *testing.T, testContext *helper.TestContext, _ map[string]k8s.Object) {
		client := helper.NewHTTPTestClient(t, storage.ServiceType_SCANNER_SERVICE)
		req, err := http.NewRequest(http.MethodGet, "https://localhost:8443/scanner/definitions", nil)
		require.NoError(t, err)

		// Sensor may not be yet in the ready state and may reject the API call
		assert.EventuallyWithT(t, func(collect *assert.CollectT) {
			t.Logf("Attempting GET call to https://localhost:8443/scanner/definitions")
			resp, err := client.Do(req)
			assert.NoError(collect, err)
			require.Equal(collect, http.StatusOK, resp.StatusCode, "response: %v", resp)
		}, time.Second*5, time.Millisecond*500, "Could not fetch scanner definitions")
	}))
}
