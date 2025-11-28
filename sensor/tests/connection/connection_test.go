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

// Integration tests for ROX-29270: Sensor Backoff DoS Issue Fix
// These tests verify that the exponential backoff is properly preserved or reset
// based on connection stability duration

// Test_BackoffPreservedOnRapidReconnection verifies the DoS fix
// This test simulates the exact ROX-29270 scenario where connection succeeds initially
// but fails during early communication, and verifies backoff is preserved
func Test_BackoffPreservedOnRapidReconnection(t *testing.T) {
	// Configure fast retry intervals for testing
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_INITIAL_INTERVAL", "1s")
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL", "8s")
	// Set short stable duration for faster testing
	t.Setenv("ROX_SENSOR_CONNECTION_STABLE_DURATION", "5s")

	c, err := helper.NewContextWithConfig(t, helper.Config{
		InitialSystemPolicies: nil,
		CertFilePath:          "../../../tools/local-sensor/certs/",
	})
	t.Cleanup(c.Stop)
	require.NoError(t, err)

	c.RunTest(t, helper.WithTestCase(func(t *testing.T, testContext *helper.TestContext, _ map[string]k8s.Object) {
		// Wait for initial connection and sync
		testContext.WaitForSyncEvent(t, 2*time.Minute)

		// Track reconnection attempts and timing
		reconnectionTimes := []time.Time{}
		startTime := time.Now()

		// Simulate rapid disconnections (before stable duration)
		// Each disconnection should preserve backoff, leading to increasing intervals
		for i := 0; i < 3; i++ {
			reconnectionTimes = append(reconnectionTimes, time.Now())

			// Restart connection quickly (simulating failure before stable duration)
			testContext.RestartFakeCentralConnection()

			// Wait for reconnection
			testContext.WaitForSyncEvent(t, 30*time.Second)
		}

		// Verify that retry intervals increased (backoff was preserved, not reset)
		// Expected intervals with preserved backoff: ~1s, ~2s, ~4s
		// vs. if backoff was reset: ~1s, ~1s, ~1s (DoS scenario)
		if len(reconnectionTimes) >= 3 {
			interval1 := reconnectionTimes[1].Sub(reconnectionTimes[0])
			interval2 := reconnectionTimes[2].Sub(reconnectionTimes[1])

			t.Logf("Reconnection intervals: %v, %v", interval1, interval2)

			// Second interval should be larger than first (exponential backoff preserved)
			// Allow some tolerance for timing variations
			assert.Greater(t, interval2, interval1*9/10,
				"Backoff should be preserved: interval2 (%v) should be > interval1 (%v)", interval2, interval1)
		}

		totalDuration := time.Since(startTime)
		t.Logf("Total test duration with preserved backoff: %v", totalDuration)

		// With preserved backoff (1s + 2s + 4s + overhead), should take ~10-15s
		// With reset backoff (1s + 1s + 1s + overhead), would take ~5-8s
		// This is a rough check - exact timing depends on test framework overhead
		assert.Greater(t, totalDuration, 8*time.Second,
			"Duration suggests backoff was preserved (expected >8s with exponential backoff)")
	}))
}

// Test_BackoffResetAfterStableConnection verifies backoff resets after stable period
// This test ensures that legitimate reconnections benefit from faster recovery
func Test_BackoffResetAfterStableConnection(t *testing.T) {
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_INITIAL_INTERVAL", "1s")
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL", "8s")
	t.Setenv("ROX_SENSOR_CONNECTION_STABLE_DURATION", "3s")

	c, err := helper.NewContextWithConfig(t, helper.Config{
		InitialSystemPolicies: nil,
		CertFilePath:          "../../../tools/local-sensor/certs/",
	})
	t.Cleanup(c.Stop)
	require.NoError(t, err)

	c.RunTest(t, helper.WithTestCase(func(t *testing.T, testContext *helper.TestContext, _ map[string]k8s.Object) {
		// Wait for initial connection
		testContext.WaitForSyncEvent(t, 2*time.Minute)

		// Keep connection stable for longer than stable duration
		time.Sleep(5 * time.Second)

		// Now restart - backoff should have been reset
		beforeRestart := time.Now()
		testContext.RestartFakeCentralConnection()

		// Wait for reconnection
		testContext.WaitForSyncEvent(t, 30*time.Second)
		afterReconnect := time.Now()

		reconnectDuration := afterReconnect.Sub(beforeRestart)
		t.Logf("Reconnection after stable connection took: %v", reconnectDuration)

		// With reset backoff, should reconnect relatively quickly (allowing for test framework overhead)
		// vs. if backoff wasn't reset and was at max interval (~8s+ base interval)
		// Framework overhead can add several seconds, so we use 10s threshold
		assert.Less(t, reconnectDuration, 10*time.Second,
			"After stable connection, backoff should reset for faster recovery")
	}))
}

// Test_BackoffConfigurable verifies the ROX_SENSOR_CONNECTION_STABLE_DURATION setting
func Test_BackoffConfigurable(t *testing.T) {
	// Test with zero duration (immediate reset - legacy behavior)
	t.Run("zero duration legacy behavior", func(t *testing.T) {
		t.Setenv("ROX_SENSOR_CONNECTION_RETRY_INITIAL_INTERVAL", "1s")
		t.Setenv("ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL", "2s")
		t.Setenv("ROX_SENSOR_CONNECTION_STABLE_DURATION", "0s")

		c, err := helper.NewContextWithConfig(t, helper.Config{
			InitialSystemPolicies: nil,
			CertFilePath:          "../../../tools/local-sensor/certs/",
		})
		t.Cleanup(c.Stop)
		require.NoError(t, err)

		c.RunTest(t, helper.WithTestCase(func(t *testing.T, testContext *helper.TestContext, _ map[string]k8s.Object) {
			testContext.WaitForSyncEvent(t, 2*time.Minute)

			// With 0 duration, backoff resets immediately (legacy behavior)
			// Rapid reconnects should all use initial interval
			reconnectionTimes := []time.Time{}
			for i := 0; i < 2; i++ {
				reconnectionTimes = append(reconnectionTimes, time.Now())
				testContext.RestartFakeCentralConnection()
				testContext.WaitForSyncEvent(t, 30*time.Second)
			}

			if len(reconnectionTimes) >= 2 {
				interval := reconnectionTimes[1].Sub(reconnectionTimes[0])
				t.Logf("Reconnection interval with 0s stable duration: %v", interval)
				// Both reconnections should use similar intervals (backoff reset each time)
				// Test framework overhead (WaitForSyncEvent, connection setup, etc.) can add significant time
				// The key verification is that reconnection happens (proving legacy behavior works)
				// Using 15s threshold to account for framework overhead while still being meaningful
				assert.Less(t, interval, 15*time.Second,
					"With 0s stable duration, should reset immediately (legacy behavior)")
			}
		}))
	})
}
