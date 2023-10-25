package clientreconciliation

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sensor/hash"
	"github.com/stackrox/rox/sensor/tests/helper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

var (
	resourceCases = map[string]helper.K8sResourceInfo{
		"Deployment": {
			Kind:      "Deployment",
			YamlFile:  "deployment/create.yaml",
			Name:      "nginx-deployment",
			PatchFile: "deployment/patch.json",
		},
	}
	checkType = map[string]helper.GetLastMessageTypeMatcher{
		"Deployment": func(m *central.MsgFromSensor) bool {
			return m.GetEvent().GetDeployment() != nil
		},
	}
)

type resourceDef struct {
	uid      string
	hash     uint64
	deleteFn func() error
	obj      k8s.Object
}

func Test_SensorReconciles(t *testing.T) {
	t.Setenv(features.PreventSensorRestartOnDisconnect.EnvVar(), "true")
	if !features.PreventSensorRestartOnDisconnect.Enabled() {
		t.Skip("Skip tests when ROX_PREVENT_SENSOR_RESTART_ON_DISCONNECT is disabled")
		t.SkipNow()
	}

	t.Setenv(features.SensorReconciliationOnReconnect.EnvVar(), "true")

	t.Setenv("ROX_RESYNC_DISABLED", "true")
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_INITIAL_INTERVAL", "2s")
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL", "10s")

	cfg := helper.DefaultCentralConfig()
	cfg.SendDeduperState = true
	c, err := helper.NewContextWithConfig(t, cfg)

	require.NoError(t, err)

	hasher := hash.NewHasher()

	c.RunTest(t, helper.WithTestCase(func(t *testing.T, testContext *helper.TestContext, _ map[string]k8s.Object) {
		ctx := context.Background()
		testContext.WaitForSyncEvent(t, 2*time.Minute)

		require.NoError(t, err)

		resourceMap := make(map[string]*resourceDef, len(resourceCases))

		/********************
		* TESTING CREATES
		**********************/

		for name, info := range resourceCases {
			obj := helper.ObjByKind(info.Kind)
			deleteFn, err := c.ApplyResourceAndWait(ctx, t, helper.DefaultNamespace, &info, obj, nil)
			require.NoError(t, err)
			uid := string(obj.GetUID())

			resourceMap[name] = &resourceDef{
				uid:      uid,
				deleteFn: deleteFn,
				obj:      obj,
			}
		}

		// After a deployment is created, multiple updates will happen (e.g. pod update events)
		// this sleep timer will assure that deployments are at their final state when the test runs
		time.Sleep(5 * time.Second)

		c.StopCentralGRPC()

		// Populate resource map with the latest version of messages received from central
		for name, resource := range resourceMap {
			// Wait for message in central
			messagesBeforeStopping := c.GetFakeCentral().GetAllMessages()
			lastEvent := helper.GetLastMessageWithEventIDAndType(messagesBeforeStopping, resource.uid, checkType[name])

			h, ok := hasher.HashEvent(lastEvent.GetEvent())

			require.Truef(t, ok, "Unable to hash event: %v", lastEvent.GetEvent())

			r, ok := resourceMap[name]
			require.True(t, ok)
			r.hash = h
			t.Logf("[=== TEST ===] Got hash for event %s %+v", name, r)
		}

		resourceHashes := makeResourceHashes(resourceMap)
		c.SetCentralDeduperState(central.DeduperState{
			ResourceHashes: resourceHashes,
			Current:        1,
			Total:          1,
		})

		c.StartFakeGRPC()

		testContext.WaitForSyncEvent(t, 2*time.Minute)

		// No events should've been received for the resources applied, since they are all in the deduper state
		for name, def := range resourceMap {
			require.Falsef(t, c.CheckEventReceived(def.uid, central.ResourceAction_SYNC_RESOURCE, checkType[name]), "Resource of type %s should not have been received during SYNC", name)
		}

		t.Logf("!!! Create tests passed. Starting update tests.")

		/********************
		* TESTING UPDATES
		**********************/

		for name, info := range resourceCases {
			resource, ok := resourceMap[name]
			require.True(t, ok)
			c.PatchResource(ctx, t, resource.obj, &info)
			assert.Eventually(t, func() bool {
				return c.CheckEventReceived(resource.uid, central.ResourceAction_UPDATE_RESOURCE, checkType[name])
			}, time.Minute, time.Second)
		}

		c.RestartFakeCentralConnection()

		testContext.WaitForSyncEvent(t, 2*time.Minute)

		// All events should've been patched, meaning that all resources should be received with SYNC action
		for name, def := range resourceMap {
			assert.Truef(t, c.CheckEventReceived(def.uid, central.ResourceAction_SYNC_RESOURCE, checkType[name]), "Resource of type %s should have been received during SYNC", name)
		}

		/********************
		* TESTING DELETES
		**********************/

		for name, resource := range resourceMap {
			require.NoErrorf(t, resource.deleteFn(), "Deleting resource %s", name)
			// This is just to make sure that sensor fully processed delete events before we restart the connection
			assert.Eventually(t, func() bool {
				return c.CheckEventReceived(resource.uid, central.ResourceAction_REMOVE_RESOURCE, checkType[name])
			}, time.Minute, time.Second)
		}

		c.RestartFakeCentralConnection()

		testContext.WaitForSyncEvent(t, 2*time.Minute)

		// All resources should've been deleted, meaning that after SYNC, we should eventually see all deletes arriving
		// due to the hash reconciliator in Sensor.
		for name, def := range resourceMap {
			require.Eventuallyf(t, func() bool {
				return c.CheckEventReceived(def.uid, central.ResourceAction_REMOVE_RESOURCE, checkType[name])
			}, time.Minute, time.Second, "Should receive delete for resource %s", name)
		}

	}))
}

func makeResourceHashes(in map[string]*resourceDef) map[string]uint64 {
	result := make(map[string]uint64, len(in))
	for name, def := range in {
		result[makeKey(name, def.uid)] = def.hash
	}
	return result
}

func makeKey(name, uid string) string {
	return fmt.Sprintf("%s:%s", name, uid)
}
