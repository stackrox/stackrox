package k8sreconciliation

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
	appsV1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

var (
	NginxUnchanged          = helper.K8sResourceInfo{Kind: "Deployment", YamlFile: "nginx.yaml", Name: "nginx-deployment"}
	NginxDeletedWhenOffline = helper.K8sResourceInfo{Kind: "Deployment", YamlFile: "nginx2.yaml", Name: "nginx-deployment-2"}
	NginxCreatedWhenOffline = helper.K8sResourceInfo{Kind: "Deployment", YamlFile: "nginx3.yaml", Name: "nginx-deployment-3"}
	NginxUpdatedWhenOffline = helper.K8sResourceInfo{Kind: "Deployment", YamlFile: "nginx4.yaml", Name: "nginx-deployment-4",
		PatchFile: "nginx4-patch.json"}
	NetpolBlockEgress = helper.K8sResourceInfo{Kind: "NetworkPolicy", YamlFile: "netpol-block-egress.yaml", Name: "block-all-egress"}

	checkType = map[string]helper.GetLastMessageTypeMatcher{
		"Deployment": func(m *central.MsgFromSensor) bool {
			return m.GetEvent().GetDeployment() != nil
		},
		"NetworkPolicy": func(m *central.MsgFromSensor) bool {
			return m.GetEvent().GetNetworkPolicy() != nil
		},
	}
)

type resourceDef struct {
	kind     string
	uid      string
	hash     uint64
	deleteFn func() error
	obj      k8s.Object
}

func Test_SensorReconcilesKubernetesEvents(t *testing.T) {
	t.Setenv(features.PreventSensorRestartOnDisconnect.EnvVar(), "true")
	if !features.PreventSensorRestartOnDisconnect.Enabled() {
		t.Skip("Skip tests when ROX_PREVENT_SENSOR_RESTART_ON_DISCONNECT is disabled")
		t.SkipNow()
	}
	t.Setenv("ROX_RESYNC_DISABLED", "true")
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_INITIAL_INTERVAL", "1s")
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL", "2s")

	cfg := helper.DefaultCentralConfig()
	cfg.SendDeduperState = true
	c, err := helper.NewContextWithConfig(t, cfg)
	require.NoError(t, err)

	hasher := hash.NewHasher()

	// Timeline of the events in this test:
	// 1) Create deployments NginxUnchanged, NginxUpdated, NginxDeletedWhenOffline
	// 2) Create NetworkPolicy NetpolBlockEgress
	// 3) gRPC Connection interrupted
	// 4) Delete deployment NginxDeletedWhenOffline
	// 5) Create deployment NginxCreatedWhenOffline
	// 6) Update deployment NginxUpdatedWhenOffline
	// 7) gRPC Connection re-established
	// 8) Deduper State received with (NginxUnchanged, NginxUpdated, NginxDeletedWhenOffline, NetpolBlockEgress)
	// 9) Sensor will transmit three messages:
	//  - SYNC NginxCreatedWhenOffline
	//  - SYNC NginxUpdatedWhenOffline
	//  - ResourcesSyncedEvent with IDs: NginxUnchaged and NetpolBlockEgress
	//
	// Using a NetworkPolicy here will make sure that no deployments that were removed while the connection
	// was down will be reprocessed and sent when the NetworkPolicy event gets resynced.
	c.RunTest(t, helper.WithTestCase(func(t *testing.T, testContext *helper.TestContext, _ map[string]k8s.Object) {
		ctx := context.Background()

		testContext.WaitForSyncEvent(t, 2*time.Minute)

		resourceMap := map[string]*resourceDef{}

		initialResources := []helper.K8sResourceInfo{NetpolBlockEgress, NginxUnchanged, NginxDeletedWhenOffline, NginxUpdatedWhenOffline}
		for _, resourceToApply := range initialResources {
			obj := helper.ObjByKind(resourceToApply.Kind)
			resource := resourceToApply
			deleteFn, err := c.ApplyResourceAndWait(ctx, t, helper.DefaultNamespace, &resource, obj, nil)
			require.NoError(t, err)
			uid := string(obj.GetUID())

			resourceMap[resourceToApply.YamlFile] = &resourceDef{
				uid:      uid,
				deleteFn: deleteFn,
				obj:      obj,
				kind:     resourceToApply.Kind,
			}
		}

		testContext.StopCentralGRPC()

		messagesBeforeStopping := c.GetFakeCentral().GetAllMessages()
		// Populate resource map with the latest version of messages received from central
		for name, resource := range resourceMap {
			// Wait for message in central
			fn, ok := checkType[resource.kind]
			require.Truef(t, ok, "no checkType function for kind %s create one in the tests", resource.kind)
			lastEvent := helper.GetLastMessageWithEventIDAndType(messagesBeforeStopping, resource.uid, fn)
			require.NotNilf(t, lastEvent, "Should have received an event with ID %s", resource.uid)

			h, ok := hasher.HashEvent(lastEvent.GetEvent())

			require.Truef(t, ok, "Unable to hash event: %v", lastEvent.GetEvent())

			r, ok := resourceMap[name]
			require.True(t, ok)
			r.hash = h
		}

		resourceHashes := makeResourceHashes(resourceMap)
		c.SetCentralDeduperState(&central.DeduperState{
			ResourceHashes: resourceHashes,
			Current:        1,
			Total:          1,
		})

		obj := &appsV1.Deployment{}
		_, err = c.ApplyResource(ctx, t, helper.DefaultNamespace, &NginxCreatedWhenOffline, obj, nil)
		require.NoError(t, err)

		require.NoError(t, resourceMap[NginxDeletedWhenOffline.YamlFile].deleteFn())

		c.PatchResource(ctx, t, resourceMap[NginxUpdatedWhenOffline.YamlFile].obj, &NginxUpdatedWhenOffline)

		testContext.StartFakeGRPC()

		archived := testContext.ArchivedMessages()
		require.Len(t, archived, 1)
		deploymentMessageInArchive(t, archived[0], helper.DefaultNamespace, NginxUnchanged.Name)

		// Wait for sync event to be sent
		synced := testContext.WaitForSyncEvent(t, 2*time.Minute)

		// Synced message should have stub for unchanged NginxUnchanged and NetpolBlockEgress
		assert.Contains(t, synced.UnchangedIds, fmt.Sprintf("Deployment:%s", resourceMap[NginxUnchanged.YamlFile].uid))
		assert.Contains(t, synced.UnchangedIds, fmt.Sprintf("NetworkPolicy:%s", resourceMap[NetpolBlockEgress.YamlFile].uid))

		// Expect the following state to be communicated:
		testContext.FirstDeploymentReceivedWithAction(t, NginxUpdatedWhenOffline.Name, central.ResourceAction_SYNC_RESOURCE)
		testContext.FirstDeploymentReceivedWithAction(t, NginxCreatedWhenOffline.Name, central.ResourceAction_SYNC_RESOURCE)
		testContext.EventIDNotReceived(t, resourceMap[NginxDeletedWhenOffline.YamlFile].uid, checkType["Deployment"])
		testContext.EventIDNotReceived(t, resourceMap[NginxUnchanged.YamlFile].uid, checkType["Deployment"])
		testContext.EventIDNotReceived(t, resourceMap[NetpolBlockEgress.YamlFile].uid, checkType["NetworkPolicy"])
	}))

}

func deploymentMessageInArchive(t *testing.T, messages []*central.MsgFromSensor, namespace, deploymentName string) {
	t.Logf("%d messages in archive", len(messages))
	for _, m := range messages {
		dep := m.GetEvent().GetDeployment()
		if dep != nil && dep.GetNamespace() == namespace && dep.GetName() == deploymentName && m.GetEvent().GetAction() != central.ResourceAction_SYNC_RESOURCE {
			return
		}
	}
	t.Errorf("could not find deployment %s:%s in archived messages", namespace, deploymentName)
}

func makeResourceHashes(in map[string]*resourceDef) map[string]uint64 {
	result := make(map[string]uint64, len(in))
	for _, def := range in {
		result[makeKey(def.kind, def.uid)] = def.hash
	}
	return result
}

func makeKey(name, uid string) string {
	return fmt.Sprintf("%s:%s", name, uid)
}
