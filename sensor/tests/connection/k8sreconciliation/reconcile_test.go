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

	c.RunTest(helper.WithTestCase(func(t *testing.T, testContext *helper.TestContext, _ map[string]k8s.Object) {
		ctx := context.Background()

		testContext.WaitForSyncEvent(2 * time.Minute)
		_, err = c.ApplyResourceAndWaitNoObject(ctx, helper.DefaultNamespace, NginxDeployment1, nil)

		testContext.StopCentralGRPC()

		obj := &appsV1.Deployment{}
		_, err = c.ApplyResource(ctx, helper.DefaultNamespace, &NginxDeployment2, obj, nil)
		require.NoError(t, err)

		testContext.StartFakeGRPC()

		archived := testContext.ArchivedMessages()
		require.Len(t, archived, 1)
		deploymentMessageInArchive(t, archived[0], helper.DefaultNamespace, NginxDeployment1.Name)

		testContext.WaitForSyncEvent(2 * time.Minute)
		testContext.DeploymentActionReceived(NginxDeployment1.Name, central.ResourceAction_SYNC_RESOURCE)
		testContext.DeploymentActionReceived(NginxDeployment2.Name, central.ResourceAction_SYNC_RESOURCE)
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
