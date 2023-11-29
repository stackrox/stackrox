package alerts

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/tests/helper"
	"github.com/stackrox/rox/sensor/testutils"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

var (
	DeploymentWithViolation = helper.K8sResourceInfo{Kind: "Deployment", YamlFile: "nginx.yaml", Name: "nginx-deployment"}
)

func Test_AlertsAreSentAfterConnectionRestart(t *testing.T) {
	t.Setenv(features.PreventSensorRestartOnDisconnect.EnvVar(), "true")
	if !features.PreventSensorRestartOnDisconnect.Enabled() {
		t.Skip("Skip tests when ROX_PREVENT_SENSOR_RESTART_ON_DISCONNECT is disabled")
		t.SkipNow()
	}

	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_INITIAL_INTERVAL", "1s")
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL", "2s")

	config := helper.DefaultCentralConfig()
	var err error
	config.InitialSystemPolicies, err = testutils.GetPoliciesFromFile("./data/policies.json")
	require.NoError(t, err)

	c, err := helper.NewContextWithConfig(t, config)
	require.NoError(t, err)

	c.RunTest(t, helper.WithTestCase(func(t *testing.T, testContext *helper.TestContext, _ map[string]k8s.Object) {
		ctx := context.Background()

		testContext.WaitForSyncEvent(t, 2*time.Minute)

		_, err = c.ApplyResourceAndWaitNoObject(ctx, t, helper.DefaultNamespace, DeploymentWithViolation, nil)
		require.NoError(t, err)

		// After first deployment, should see an alert for deployment
		testContext.LastViolationState(t, DeploymentWithViolation.Name, hasRequiredLabelAlert, "Deployment should have alerts")

		// We need to wait some virtual time until all the deployment updates happen before restarting. Otherwise,
		// the deployment will continue to receive updates (e.g. image SHA update) and the test will pass even if
		// the deduper is not reset
		time.Sleep(30 * time.Second)

		// Simulate a blip in the Network connection
		testContext.RestartFakeCentralConnection()

		// Wait for reconciliation to finish
		testContext.WaitForSyncEvent(t, 2*time.Minute)

		// Should see the alert *again* on a connection restart
		testContext.LastViolationState(t, DeploymentWithViolation.Name, hasRequiredLabelAlert, "Deployment should have alerts")
	}))

}

func hasRequiredLabelAlert(alertResults *central.AlertResults) error {
	alerts := alertResults.GetAlerts()
	if len(alerts) != 1 {
		return errors.Errorf("expected 1 alert to return but received: %d", len(alerts))
	}

	if alerts[0].Policy.Name != "Required Label: Owner/Team" {
		return errors.Errorf("expected alert with name 'Required Label: Owner/Team' but is '%s' instead", alerts[0].Policy.Name)
	}

	return nil
}
