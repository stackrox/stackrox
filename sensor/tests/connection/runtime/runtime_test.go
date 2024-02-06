package runtime

import (
	"context"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	net2 "github.com/stackrox/rox/pkg/net"
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
	config := helper.DefaultCentralConfig()
	config.RealCerts = false
	config.InitialSystemPolicies, err = testutils.GetPoliciesFromFile("../../data/runtime-policies.json")
	require.NoError(t, err)
	if config.RealCerts {
		config.CertFilePath = "../../../../tmp"
	}
	c, err := helper.NewContextWithConfig(t, config)
	require.NoError(t, err)

	var fakeCollector *collector.FakeCollector
	if !config.RealCerts {
		fakeCollector = collector.NewFakeCollector(collector.WithDefaultConfig().WithCertsPath(config.CertFilePath))
		require.NoError(t, fakeCollector.Start())
	}

	c.RunTest(t, helper.WithTestCase(func(t *testing.T, testContext *helper.TestContext, _ map[string]k8s.Object) {
		ctx := context.Background()

		testContext.WaitForSyncEvent(t, 2*time.Minute)

		// Wait for collector to connect
		time.Sleep(30 * time.Second)
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

		// If we are using fake collector
		if !config.RealCerts {
			nginxContainerIds := testContext.GetContainerIdsFromDeployment(nginxObj)
			require.Len(t, nginxContainerIds, 1)
			deployIP := testContext.GetIPsFromDeployment(nginxObj)
			require.Len(t, deployIP, 1)
			srvIP := testContext.GetIPFromService(srvObj)
			require.NotEqual(t, "", srvIP)
			talkContainerIds := testContext.GetContainerIdsFromPod(talkObj)
			require.Len(t, talkContainerIds, 1)
			podIP := testContext.GetIPFromPod(talkObj)
			require.NotEqual(t, "", podIP)

			fakeCollector.SendFakeSignal(&sensor.SignalStreamMessage{
				Msg: &sensor.SignalStreamMessage_Signal{
					Signal: &v1.Signal{
						Signal: &v1.Signal_ProcessSignal{
							ProcessSignal: &storage.ProcessSignal{
								ContainerId: talkContainerIds[0],
								Name:        "curl",
							},
						},
					},
				},
			})
			fakeCollector.SendFakeNetworkFlow(&sensor.NetworkConnectionInfoMessage{
				Msg: &sensor.NetworkConnectionInfoMessage_Info{
					Info: &sensor.NetworkConnectionInfo{
						UpdatedConnections: []*sensor.NetworkConnection{
							{
								SocketFamily: sensor.SocketFamily_SOCKET_FAMILY_UNKNOWN,
								Protocol:     storage.L4Protocol_L4_PROTOCOL_TCP,
								Role:         sensor.ClientServerRole_ROLE_CLIENT,
								ContainerId:  talkContainerIds[0],
								LocalAddress: &sensor.NetworkAddress{
									AddressData: nil,
									IpNetwork:   nil,
									Port:        0,
								},
								RemoteAddress: &sensor.NetworkAddress{
									AddressData: net2.ParseIP(srvIP).AsNetIP(),
									IpNetwork:   net2.ParseIP(srvIP).AsNetIP(),
									Port:        80,
								},
							},
						},
					},
				},
			})

			var nginxContainers []string
			for _, conn := range nginxContainerIds {
				nginxContainers = append(nginxContainers, conn...)
			}
			fakeCollector.SendFakeNetworkFlow(&sensor.NetworkConnectionInfoMessage{
				Msg: &sensor.NetworkConnectionInfoMessage_Info{
					Info: &sensor.NetworkConnectionInfo{
						UpdatedConnections: []*sensor.NetworkConnection{
							{
								SocketFamily: sensor.SocketFamily_SOCKET_FAMILY_UNKNOWN,
								Protocol:     storage.L4Protocol_L4_PROTOCOL_TCP,
								Role:         sensor.ClientServerRole_ROLE_SERVER,
								ContainerId:  nginxContainers[0],
								LocalAddress: &sensor.NetworkAddress{
									AddressData: nil,
									IpNetwork:   nil,
									Port:        80,
								},
								RemoteAddress: &sensor.NetworkAddress{
									AddressData: net2.ParseIP(podIP).AsNetIP(),
									IpNetwork:   net2.ParseIP(podIP).AsNetIP(),
									Port:        0,
								},
							},
						},
					},
				},
			})
		}
		// We need to wait here at least 30s to make sure the network flows are processed
		time.Sleep(80 * time.Second)

		require.NoError(t, deleteTalk())
		require.NoError(t, testContext.WaitForResourceDeleted(talkObj))
		require.NoError(t, deleteNginx())
		require.NoError(t, testContext.WaitForResourceDeleted(nginxObj))
		require.NoError(t, deleteService())

		testContext.StartFakeGRPC()
		testContext.WaitForSyncEvent(t, 2*time.Minute)
		// Wait for the updates to be sent
		time.Sleep(30 * time.Second)

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
