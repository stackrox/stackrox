package connection

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/sensor/tests/helper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

var (
	NginxDeployment1 = helper.K8sResourceInfo{Kind: "Deployment", YamlFile: "nginx.yaml", Name: "nginx-deployment"}
	NginxDeployment2 = helper.K8sResourceInfo{Kind: "Deployment", YamlFile: "nginx2.yaml", Name: "nginx-deployment-2"}
)

func Test_SensorReconnects(t *testing.T) {
	if buildinfo.ReleaseBuild {
		t.Skipf("Don't run test in release mode: feature flag cannot be enabled")
	}

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
		// If this test becomes too flaky before the tasks above are implemented, the following mitigations could be
		// implemented:
		//  1) Increase sleep time.
		//  2) Add a signal in sensor (that can be read by the test harness) that triggers when connection is established.

		// This is used as initial signal that Sensor is fully operational, and has communicated that NginxDeployment1
		// is up to Central.
		_, err = c.ApplyResourceNoObject(context.Background(), helper.DefaultNamespace, NginxDeployment1, nil)
		require.NoError(t, err)

		// Stop fake central gRPC server and create a new one immediately after.
		testContext.RestartFakeCentralConnection()
		time.Sleep(2 * time.Second)
		assert.False(t, testContext.SensorStopped())

		// We applied the resource _after_ Sensor restarted. Now we should check that this deployment will be sent to Central.
		_, err = c.ApplyResourceNoObject(context.Background(), helper.DefaultNamespace, NginxDeployment2, nil)
		require.NoError(t, err)
	}))
}
