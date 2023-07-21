package k8sreconciliation

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/tests/helper"
	"github.com/stretchr/testify/require"
	appsV1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

var (
	NginxDeployment1 = helper.K8sResourceInfo{Kind: "Deployment", YamlFile: "nginx.yaml", Name: "nginx-deployment"}
	NginxDeployment2 = helper.K8sResourceInfo{Kind: "Deployment", YamlFile: "nginx2.yaml", Name: "nginx-deployment-2"}
	NginxDeployment3 = helper.K8sResourceInfo{Kind: "Deployment", YamlFile: "nginx3.yaml", Name: "nginx-deployment-3"}

	NetpolBlockEgress = helper.K8sResourceInfo{Kind: "NetworkPolicy", YamlFile: "netpol-block-egress.yaml", Name: "block-all-egress"}
)

func Test_SensorReconcilesKubernetesEvents(t *testing.T) {
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

	// Timeline of the events in this test:
	// 1) Create deployment Nginx1
	// 2) Create deployment Nginx2
	// 3) Create NetworkPolicy block-all-egress
	// 4) gRPC Connection interrupted
	// 5) Delete deployment Nginx2
	// 6) Create deployment Nginx3
	// 7) gRPC Connection re-established
	// 8) Sensor transmits current state with SYNC event type:
	//  Deployment 		Nginx1
	//  Deployment 		Nginx3
	//  NetworkPolicy 	block-all-egress
	//
	// Using a NetworkPolicy here will make sure that no deployments that were removed while the connection
	// was down will be used for
	//
	c.RunTest(helper.WithTestCase(func(t *testing.T, testContext *helper.TestContext, _ map[string]k8s.Object) {
		ctx := context.Background()

		testContext.WaitForSyncEvent(2 * time.Minute)
		_, err = c.ApplyResourceAndWaitNoObject(ctx, helper.DefaultNamespace, NginxDeployment1, nil)
		require.NoError(t, err)
		deleteDeployment2, err := c.ApplyResourceAndWaitNoObject(ctx, helper.DefaultNamespace, NginxDeployment2, nil)
		require.NoError(t, err)

		_, err = c.ApplyResourceAndWaitNoObject(ctx, helper.DefaultNamespace, NetpolBlockEgress, nil)
		require.NoError(t, err)

		testContext.StopCentralGRPC()

		obj := &appsV1.Deployment{}
		_, err = c.ApplyResource(ctx, helper.DefaultNamespace, &NginxDeployment3, obj, nil)
		require.NoError(t, err)

		require.NoError(t, deleteDeployment2())

		testContext.StartFakeGRPC()

		archived := testContext.ArchivedMessages()
		require.Len(t, archived, 1)
		deploymentMessageInArchive(t, archived[0], helper.DefaultNamespace, NginxDeployment1.Name)

		// Wait for sync event to be sent, then expect the following state to be transmitted:
		//   SYNC nginx-deployment-1
		//   SYNC nginx-deployment-3
		//   No event for nginx-deployment-2 (was deleted while connection was down)
		// This reconciliation state will make Central delete Nginx2, keep Nginx1 and create Nginx3
		testContext.WaitForSyncEvent(2 * time.Minute)
		testContext.FirstDeploymentReceivedWithAction(NginxDeployment1.Name, central.ResourceAction_SYNC_RESOURCE)
		testContext.FirstDeploymentReceivedWithAction(NginxDeployment3.Name, central.ResourceAction_SYNC_RESOURCE)

		// This assertion will fail if events are not properly cleared from the internal queues and in-memory stores
		testContext.DeploymentNotReceived(NginxDeployment2.Name)
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
