package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/debugger/collector"
	"github.com/stackrox/rox/sensor/tests/helper"
	"github.com/stackrox/rox/sensor/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

var (
	NginxDeployment = helper.K8sResourceInfo{Kind: "Deployment", YamlFile: "nginx.yaml", Name: "nginx-deployment"}
	NginxService    = helper.K8sResourceInfo{Kind: "Service", YamlFile: "nginx-service.yaml", Name: "nginx-service"}
	TalkPod         = helper.K8sResourceInfo{Kind: "Pod", YamlFile: "talk.yaml", Name: "talk"}

	processIndicatorPolicyName = "test-pi-curl"
	networkFlowPolicyName      = "test-flow"
)

func Test_SensorIntermediateRuntimeEvents(t *testing.T) {
	t.Setenv(features.PreventSensorRestartOnDisconnect.EnvVar(), "true")
	t.Setenv(features.SensorReconciliationOnReconnect.EnvVar(), "true")
	t.Setenv(features.SensorCapturesIntermediateEvents.EnvVar(), "true")

	t.Setenv(env.ConnectionRetryInitialInterval.EnvVar(), "1s")
	t.Setenv(env.ConnectionRetryMaxInterval.EnvVar(), "2s")

	var err error
	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()
	config := helper.DefaultConfig()
	config.RealCerts = helper.UseRealCollector.BooleanSetting()
	config.InitialSystemPolicies, err = testutils.GetPoliciesFromFile("../../data/runtime-policies.json")
	require.NoError(t, err)

	flowC := make(chan *sensor.NetworkConnectionInfoMessage, 1000)
	config.NetworkFlowTraceWriter = helper.NewNetworkFlowTraceWriter(ctx, flowC)
	signalC := make(chan *sensor.SignalStreamMessage, 1000)
	config.ProcessIndicatorTraceWriter = helper.NewProcessIndicatorTraceWriter(ctx, signalC)

	if config.RealCerts {
		config.CertFilePath = "tmp/"
	}
	c, err := helper.NewContextWithConfig(t, config)
	require.NoError(t, err)

	var fakeCollector *collector.FakeCollector
	if !helper.UseRealCollector.BooleanSetting() {
		fakeCollector = collector.NewFakeCollector(collector.WithDefaultConfig().WithCertsPath(config.CertFilePath))
		require.NoError(t, fakeCollector.Start())
	}

	c.RunTest(t, helper.WithTestCase(func(t *testing.T, testContext *helper.TestContext, _ map[string]k8s.Object) {
		testContext.WaitForSyncEvent(t, 2*time.Minute)

		// Wait for collector to connect
		waitIfRealCollector(30 * time.Second)
		testContext.GetFakeCentral().ClearReceivedBuffer()
		testContext.StopCentralGRPC()

		// Nginx deployment
		nginxObj := helper.ObjByKind(NginxDeployment.Kind)
		deleteNginx, err := c.ApplyResource(ctx, t, helper.DefaultNamespace, &NginxDeployment, nginxObj, nil)
		require.NoError(t, err)
		nginxUID := string(nginxObj.GetUID())

		// Nginx service
		srvObj := helper.ObjByKind(NginxService.Kind)
		deleteService, err := c.ApplyResource(ctx, t, helper.DefaultNamespace, &NginxService, srvObj, nil)
		require.NoError(t, err)

		// Talk pod
		talkObj := helper.ObjByKind(TalkPod.Kind)
		deleteTalk, err := c.ApplyResource(ctx, t, helper.DefaultNamespace, &TalkPod, talkObj, nil)
		require.NoError(t, err)
		talkUID := string(talkObj.GetUID())
		talkContainerIds := testContext.GetContainerIdsFromPod(ctx, talkObj)
		require.Len(t, talkContainerIds, 1)
		talkIP := testContext.GetIPFromPod(talkObj)
		require.NotEqual(t, "", talkIP)

		if !helper.UseRealCollector.BooleanSetting() {
			nginxPodIDs, nginxContainerIDs := c.GetContainerIdsFromDeployment(nginxObj)
			require.Len(t, nginxPodIDs, 1)
			require.Len(t, nginxContainerIDs, 1)
			require.Len(t, nginxContainerIDs[nginxPodIDs[0]], 1)
			require.NotEqual(t, "", nginxContainerIDs[nginxPodIDs[0]][0])

			nginxIP := c.GetIPFromService(srvObj)
			require.NotEqual(t, "", nginxIP)

			helper.SendSignalMessage(fakeCollector, talkContainerIds[0], "curl")
			helper.SendFlowMessage(fakeCollector,
				sensor.SocketFamily_SOCKET_FAMILY_UNKNOWN,
				storage.L4Protocol_L4_PROTOCOL_TCP,
				talkContainerIds[0],
				nginxContainerIDs[nginxPodIDs[0]][0],
				talkIP,
				nginxIP,
				80,
			)
		}
		messagesReceivedSignal := concurrency.NewErrorSignal()
		expectedNetworkFlows := []helper.ExpectedNetworkConnectionMessageFn{
			func(msg *sensor.NetworkConnectionInfoMessage) bool {
				for _, conn := range msg.GetInfo().GetUpdatedConnections() {
					if conn.Protocol == storage.L4Protocol_L4_PROTOCOL_TCP && conn.ContainerId == talkContainerIds[0] && conn.GetRemoteAddress().GetPort() == 80 {
						return true
					}
				}
				return false
			},
		}
		expectedSignals := []helper.ExpectedSignalMessageFn{
			func(msg *sensor.SignalStreamMessage) bool {
				return msg.GetSignal().GetProcessSignal().GetName() == "curl" && msg.GetSignal().GetProcessSignal().GetContainerId() == talkContainerIds[0]
			},
		}
		go helper.WaitToReceiveMessagesFromCollector(ctx, &messagesReceivedSignal,
			signalC,
			flowC,
			expectedSignals,
			expectedNetworkFlows)
		require.NoError(t, messagesReceivedSignal.Wait())

		// We need to wait here at least 30s to make sure the network flows are processed
		time.Sleep(60 * time.Second)

		require.NoError(t, deleteTalk())
		require.NoError(t, testContext.WaitForResourceDeleted(talkObj))
		require.NoError(t, deleteNginx())
		require.NoError(t, testContext.WaitForResourceDeleted(nginxObj))
		require.NoError(t, deleteService())

		testContext.StartFakeGRPC()
		testContext.WaitForSyncEvent(t, 2*time.Minute)

		msg, err := testContext.WaitForMessageWithMatcher(func(event *central.MsgFromSensor) bool {
			return event.GetEvent().GetProcessIndicator().GetDeploymentId() == talkUID &&
				event.GetEvent().GetProcessIndicator().GetSignal().GetName() == "curl"
		}, time.Minute)
		assert.NoError(t, err)
		assert.NotNil(t, msg)
		msg, err = testContext.WaitForMessageWithMatcher(func(event *central.MsgFromSensor) bool {
			for _, flow := range event.GetNetworkFlowUpdate().GetUpdated() {
				if flow.GetProps().GetSrcEntity().GetId() == talkUID && flow.GetProps().GetDstEntity().GetId() == nginxUID {
					return true
				}
			}
			return false
		}, time.Minute)
		assert.NoError(t, err)
		assert.NotNil(t, msg)
		testContext.AssertViolationStateByID(t, talkUID, helper.AssertViolationsMatch(networkFlowPolicyName), networkFlowPolicyName, false)
		testContext.AssertViolationStateByID(t, talkUID, helper.AssertViolationsMatch(processIndicatorPolicyName), processIndicatorPolicyName, false)
		testContext.AssertViolationStateByID(t, nginxUID, helper.AssertViolationsMatch(networkFlowPolicyName), networkFlowPolicyName, false)
	}))
}

func waitIfRealCollector(sleepTime time.Duration) {
	if helper.UseRealCollector.BooleanSetting() {
		time.Sleep(sleepTime)
	}
}
