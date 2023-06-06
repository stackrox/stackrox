package k8sreconciliation

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/tests/helper"
	"github.com/stretchr/testify/require"
	appsV1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

var (
	NginxDeployment1 = helper.K8sResourceInfo{Kind: "Deployment", YamlFile: "nginx.yaml", Name: "nginx-deployment"}
	NginxDeployment2 = helper.K8sResourceInfo{Kind: "Deployment", YamlFile: "nginx2.yaml", Name: "nginx-deployment-2"}
)

func Test_SensorReconcilesKubernetesEvents(t *testing.T) {
	t.Setenv("ROX_PREVENT_SENSOR_RESTART_ON_DISCONNECT", "true")
	t.Setenv("ROX_RESYNC_DISABLED", "true")
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_INITIAL_INTERVAL", "1s")
	t.Setenv("ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL", "2s")

	c, err := helper.NewContext(t)
	require.NoError(t, err)

	c.RunTest(helper.WithTestCase(func(t *testing.T, testContext *helper.TestContext, _ map[string]k8s.Object) {
		ctx, cancelFn := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancelFn()
		_, err = c.ApplyResourceNoObject(ctx, helper.DefaultNamespace, NginxDeployment1, nil)

		testContext.WaitForSyncEvent()

		testContext.StopCentralGRPC()

		obj := &appsV1.Deployment{}
		_, err = c.ApplyResource(context.Background(), helper.DefaultNamespace, &NginxDeployment2, obj, nil)
		require.NoError(t, err)

		testContext.StartFakeGRPC()

		archived := testContext.ArchivedMessages()
		require.Len(t, archived, 1)
		deploymentInMessages(t, archived[0], helper.DefaultNamespace, NginxDeployment1.Name)

		// TODO: Assert that sync event is sent and that deployment 1 and 2 is seen in current messages with SYNC type
		// testContext.WaitForSyncEvent()
		//testContext.DeploymentActionReceived(NginxDeployment1.Name, central.ResourceAction_SYNC_RESOURCE)
		//testContext.DeploymentActionReceived(NginxDeployment2.Name, central.ResourceAction_SYNC_RESOURCE)

	}))

}

func deploymentInMessages(t *testing.T, messages []*central.MsgFromSensor, namespace, deploymentName string) {
	t.Logf("%d messages in archive", len(messages))
	for _, m := range messages {
		dep := m.GetEvent().GetDeployment()
		if dep != nil && dep.GetNamespace() == namespace && dep.GetName() == deploymentName {
			return
		}
	}
	t.Errorf("could not find deployment %s:%s in messages", namespace, deploymentName)
}
