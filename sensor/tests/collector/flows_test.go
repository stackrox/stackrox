package collector

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
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
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

func Test_SensorLastSeenTimestamp(t *testing.T) {
	t.Setenv(features.PreventSensorRestartOnDisconnect.EnvVar(), "true")
	t.Setenv(features.SensorReconciliationOnReconnect.EnvVar(), "true")
	t.Setenv(features.SensorCapturesIntermediateEvents.EnvVar(), "true")

	t.Setenv(env.ConnectionRetryInitialInterval.EnvVar(), "1s")
	t.Setenv(env.ConnectionRetryMaxInterval.EnvVar(), "2s")

	t.Setenv(resources.PastClusterEntitiesMemorySize.EnvVar(), "0")
	t.Setenv("LOGLEVEL", "debug")

	var err error
	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()
	initialSystemPolicies, err := testutils.GetPoliciesFromFile("../data/runtime-policies.json")
	require.NoError(t, err)
	flowC := make(chan *sensor.NetworkConnectionInfoMessage, 1000)
	signalC := make(chan *sensor.SignalStreamMessage, 1000)
	ticker := make(chan time.Time)
	config := helper.Config{
		InitialSystemPolicies:       initialSystemPolicies,
		RealCerts:                   helper.UseRealCollector.BooleanSetting(),
		SendDeduperState:            false,
		NetworkFlowTraceWriter:      helper.NewNetworkFlowTraceWriter(ctx, flowC),
		ProcessIndicatorTraceWriter: helper.NewProcessIndicatorTraceWriter(ctx, signalC),
		CertFilePath:                "../../../tools/local-sensor/certs/",
		NetworkFlowTicker:           ticker,
	}

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
		//testContext.GetFakeCentral().ClearReceivedBuffer()
		//testContext.StopCentralGRPC()

		// Nginx deployment
		t.Log("Applying nginx deployment")
		nginxObj := helper.ObjByKind(NginxDeployment.Kind)
		deleteNginx, err := c.ApplyResource(ctx, t, helper.DefaultNamespace, &NginxDeployment, nginxObj, nil)
		require.NoError(t, err)
		nginxUID := string(nginxObj.GetUID())

		// Nginx service
		t.Log("Applying nginx service")
		srvObj := helper.ObjByKind(NginxService.Kind)
		deleteService, err := c.ApplyResource(ctx, t, helper.DefaultNamespace, &NginxService, srvObj, nil)
		require.NoError(t, err)

		// Talk pod
		t.Log("Applying talk pod")
		talkObj := helper.ObjByKind(TalkPod.Kind)
		deleteTalk, err := c.ApplyResource(ctx, t, helper.DefaultNamespace, &TalkPod, talkObj, nil)
		require.NoError(t, err)
		talkUID := string(talkObj.GetUID())
		talkContainerIds := testContext.GetContainerIdsFromPod(ctx, talkObj)
		require.Len(t, talkContainerIds, 1)
		talkIP := testContext.GetIPFromPod(talkObj)
		require.NotEqual(t, "", talkIP)

		t.Log("Ensure nginx deployment was deployed correctly")
		nginxPodIDs, nginxContainerIDs, nginxIP := getDeploymentInfo(t, c, nginxObj, srvObj)
		if !helper.UseRealCollector.BooleanSetting() {
			helper.SendSignalMessage(fakeCollector, talkContainerIds[0], "curl")
			helper.SendFlowMessage(fakeCollector,
				sensor.SocketFamily_SOCKET_FAMILY_UNKNOWN,
				storage.L4Protocol_L4_PROTOCOL_TCP,
				talkContainerIds[0],
				nginxContainerIDs[nginxPodIDs[0]][0],
				talkIP,
				nginxIP,
				80,
				900,
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

		t.Log("============= lvm flows received in the collector service")

		// We need to wait here at least 30s to make sure the network flows are processed
		// time.Sleep(60 * time.Second)
		time.Sleep(5 * time.Second)
		ticker <- time.Now()

		msg, err := testContext.WaitForMessageWithMatcher(assertFlow(t, talkUID, nginxUID), time.Minute)
		assert.NoError(t, err)
		assert.NotNil(t, msg)
		testContext.GetFakeCentral().ClearReceivedBuffer()

		//require.NoError(t, deleteTalk())
		//require.NoError(t, testContext.WaitForResourceDeleted(talkObj))
		//require.NoError(t, deleteNginx())
		//require.NoError(t, testContext.WaitForResourceDeleted(nginxObj))
		//require.NoError(t, deleteService())

		t.Log("============= lvm deployments deleted")

		if !helper.UseRealCollector.BooleanSetting() {
			helper.SendCloseFlowMessage(fakeCollector,
				sensor.SocketFamily_SOCKET_FAMILY_UNKNOWN,
				storage.L4Protocol_L4_PROTOCOL_TCP,
				talkContainerIds[0],
				nginxContainerIDs[nginxPodIDs[0]][0],
				talkIP,
				nginxIP,
				80,
				1000,
				//protocompat.TimestampNow().GetSeconds(),
			)
		}
		messagesReceivedSignal.Reset()
		go helper.WaitToReceiveMessagesFromCollector(ctx, &messagesReceivedSignal,
			signalC,
			flowC,
			nil,
			expectedNetworkFlows)
		require.NoError(t, messagesReceivedSignal.Wait())

		t.Log("============= lvm flows received in the collector service")

		//testContext.StartFakeGRPC()
		//testContext.WaitForSyncEvent(t, 2*time.Minute)
		time.Sleep(5 * time.Second)
		ticker <- time.Now()
		t.Log("============= lvm first tick")
		time.Sleep(1 * time.Second)

		//ticker <- time.Now()
		//// The enrichAndSend will not catch this connection as updated, because current conn has defined TS, while previous one had +inf ts - thus, this is not an update. But is that correct?
		//t.Log("============= lvm after second tick")
		//<-time.After(time.Second)

		//msg, err = testContext.WaitForMessageWithMatcher(func(event *central.MsgFromSensor) bool {
		//	return event.GetEvent().GetProcessIndicator().GetDeploymentId() == talkUID &&
		//		event.GetEvent().GetProcessIndicator().GetSignal().GetName() == "curl"
		//}, time.Minute)
		//assert.NoError(t, err)
		//assert.NotNil(t, msg)
		msg, err = testContext.WaitForMessageWithMatcher(assertFlow(t, talkUID, nginxUID), time.Minute)
		assert.NoError(t, err)
		assert.NotNil(t, msg)
		//testContext.AssertViolationStateByID(t, talkUID, helper.AssertViolationsMatch(networkFlowPolicyName), networkFlowPolicyName, false)
		//testContext.AssertViolationStateByID(t, talkUID, helper.AssertViolationsMatch(processIndicatorPolicyName), processIndicatorPolicyName, false)
		//testContext.AssertViolationStateByID(t, nginxUID, helper.AssertViolationsMatch(networkFlowPolicyName), networkFlowPolicyName, false)
		require.NoError(t, deleteTalk())
		require.NoError(t, testContext.WaitForResourceDeleted(talkObj))
		require.NoError(t, deleteNginx())
		require.NoError(t, testContext.WaitForResourceDeleted(nginxObj))
		require.NoError(t, deleteService())
	}))
}

func assertFlow(t *testing.T, fromID, toID string) func(*central.MsgFromSensor) bool {
	return func(event *central.MsgFromSensor) bool {
		found := false
		if len(event.GetNetworkFlowUpdate().GetUpdated()) > 0 {
			t.Log("============= lvm Flows")
			for _, flow := range event.GetNetworkFlowUpdate().GetUpdated() {
				t.Logf("============= lvm Flow (src=%s -> dest=%s) last seen: %v",
					flow.GetProps().GetSrcEntity().GetId(),
					flow.GetProps().GetDstEntity().GetId(),
					flow.GetLastSeenTimestamp())
				if flow.GetProps().GetSrcEntity().GetId() == fromID && flow.GetProps().GetDstEntity().GetId() == toID {
					found = true
				}
			}
		}
		if len(event.GetNetworkFlowUpdate().GetUpdatedEndpoints()) > 0 {
			t.Log("============= lvm Endpoints")
			for _, f := range event.GetNetworkFlowUpdate().GetUpdatedEndpoints() {
				t.Logf("============= lvm Endpoint: last active ts=%v", f.GetLastActiveTimestamp())
			}
		}
		return found
	}
}

func getDeploymentInfo(t *testing.T, c *helper.TestContext, deployment, service k8s.Object) (podIDs []string, containerIDs map[string][]string, ip string) {
	podIDs, containerIDs = c.GetContainerIdsFromDeployment(deployment)
	require.Len(t, podIDs, 1, "Expected to find 1 pod ID")
	require.Len(t, containerIDs, 1)
	require.Len(t, containerIDs[podIDs[0]], 1)
	require.NotEqual(t, "", containerIDs[podIDs[0]][0])

	ip = c.GetIPFromService(service)
	require.NotEqual(t, "", ip)
	return podIDs, containerIDs, ip
}

func waitIfRealCollector(sleepTime time.Duration) {
	if helper.UseRealCollector.BooleanSetting() {
		time.Sleep(sleepTime)
	}
}
