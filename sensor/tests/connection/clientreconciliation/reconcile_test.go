package clientreconciliation

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/tests/helper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

var (
	resourceCases = map[string]helper.K8sResourceInfo{
		"Deployment": {Kind: "Deployment", YamlFile: "deployment/create.yaml", Name: "nginx-deployment"},
	}
)

type resourceDef struct {
	uid      string
	hash     uint64
	deleteFn func() error
}

func Test_SensorReconciles(t *testing.T) {
	t.Setenv(features.PreventSensorRestartOnDisconnect.EnvVar(), "true")
	if !features.PreventSensorRestartOnDisconnect.Enabled() {
		t.Skip("Skip tests when ROX_PREVENT_SENSOR_RESTART_ON_DISCONNECT is disabled")
		t.SkipNow()
	}
	t.Setenv("ROX_RESYNC_DISABLED", "true")
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_INITIAL_INTERVAL", "1s")
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL", "2s")

	c, err := helper.NewContext(t)
	require.NoError(t, err)

	c.RunTest(t, helper.WithTestCase(func(t *testing.T, testContext *helper.TestContext, _ map[string]k8s.Object) {
		ctx := context.Background()

		testContext.WaitForSyncEvent(t, 2*time.Minute)

		require.NoError(t, err)

		var resourceMap map[string]resourceDef

		for name, info := range resourceCases {
			var obj k8s.Object
			deleteFn, err := c.ApplyResourceAndWait(ctx, t, helper.DefaultNamespace, &info, obj, nil)
			require.NoError(t, err)
			uid := string(obj.GetUID())

			// Wait for message in central
			msg, err := c.WaitForMessageWithEventID(uid, time.Minute)
			require.NoErrorf(t, err, "Waiting for event ID for resource type %s", name)

			resourceMap[name] = resourceDef{
				uid:      uid,
				deleteFn: deleteFn,
				hash:     msg.GetEvent().GetSensorHash(),
			}
		}

		deduperState := makeDeduperState(resourceMap)

		c.GetFakeCentral().EnableDeduperState(true)
		c.GetFakeCentral().SetDeduperState(deduperState)

		c.RestartFakeCentralConnection()

		testContext.WaitForSyncEvent(t, 2*time.Minute)

		// No events should've been received for the resources applied, since they are all in the deduper state
		for name, def := range resourceMap {
			assert.Falsef(t, c.CheckEventReceived(def.uid), "Resource of type %s should not have been received during SYNC", name)
		}
	}))
}

func makeDeduperState(in map[string]resourceDef) map[string]uint64 {
	result := make(map[string]uint64, len(in))
	for name, def := range in {
		result[makeKey(name, def.uid)] = def.hash
	}
	return result
}

func makeKey(name, uid string) string {
	return fmt.Sprintf("%s:%s", name, uid)
}
