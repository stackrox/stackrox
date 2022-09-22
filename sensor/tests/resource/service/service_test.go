package service

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/tests/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

var (
	NginxDeployment          = resource.YamlTestFile{Kind: "Deployment", File: "nginx.yaml"}
	NginxPod                 = resource.YamlTestFile{Kind: "Deployment", File: "nginx-pod.yaml"}
	NginxServiceClusterIP    = resource.YamlTestFile{Kind: "Service", File: "nginx-service-cluster-ip.yaml"}
	NginxServiceNodePort     = resource.YamlTestFile{Kind: "Service", File: "nginx-service-node-port.yaml"}
	NginxServiceLoadBalancer = resource.YamlTestFile{Kind: "Service", File: "nginx-service-load-balancer.yaml"}
)

func GetLastMessageWithDeploymentName(messages []*central.MsgFromSensor, n string) *central.MsgFromSensor {
	var lastMessage *central.MsgFromSensor
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].GetEvent().GetDeployment().GetName() == n {
			lastMessage = messages[i]
			break
		}
	}
	return lastMessage
}

func GetLastAlertsWithDeploymentID(messages []*central.MsgFromSensor, id string) *central.MsgFromSensor {
	var lastMessage *central.MsgFromSensor
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].GetEvent().GetAlertResults().GetDeploymentId() == id {
			lastMessage = messages[i]
			break
		}
	}
	return lastMessage
}

func assertLastDeploymentHasPortExposure(t *testing.T, messages []*central.MsgFromSensor, ports []*storage.PortConfig, alerts []*storage.Alert) {
	lastNginxDeploymentUpdate := GetLastMessageWithDeploymentName(messages, "nginx-deployment")
	lastNginxDeploymentAlerts := GetLastAlertsWithDeploymentID(messages, lastNginxDeploymentUpdate.GetEvent().GetDeployment().GetId())
	require.NotNil(t, lastNginxDeploymentUpdate, "should have found a message for nginx-deployment")
	require.NotNil(t, lastNginxDeploymentAlerts, "should have found an alert for nginx-deployment")
	deployment := lastNginxDeploymentUpdate.GetEvent().GetDeployment()
	actualAlerts := lastNginxDeploymentAlerts.GetEvent().GetAlertResults().GetAlerts()
	for _, expectedAlert := range alerts {
		foundAlert := false
		for _, actualAlert := range actualAlerts {
			if expectedAlert.GetPolicy().GetName() == actualAlert.GetPolicy().GetName() {
				assert.Equal(t, expectedAlert.GetState(), actualAlert.GetState())
				foundAlert = true
			}
		}
		assert.True(t, foundAlert, "Alert not found")
	}

	for _, expectedPort := range ports {
		foundPortConfig := false
		for _, port := range deployment.GetPorts() {
			if expectedPort.GetProtocol() == port.GetProtocol() &&
				expectedPort.GetContainerPort() == port.GetContainerPort() &&
				expectedPort.GetExposure() == port.GetExposure() {
				for _, expectedPortInfo := range expectedPort.GetExposureInfos() {
					foundPortInfo := false
					for _, portInfo := range port.GetExposureInfos() {
						if expectedPortInfo.GetServiceName() == portInfo.GetServiceName() {
							assert.Equal(t, expectedPortInfo.GetNodePort(), portInfo.GetNodePort())
							assert.Equal(t, expectedPortInfo.GetServicePort(), portInfo.GetServicePort())
							assert.Equal(t, expectedPortInfo.GetLevel(), portInfo.GetLevel())
							foundPortInfo = true
						}
					}
					assert.True(t, foundPortInfo, "PortInfo not found")
				}
				foundPortConfig = true
			}
		}
		assert.True(t, foundPortConfig, "PortConfig not found")
	}
}

type DeploymentExposureSuite struct {
	testContext *resource.TestContext
	suite.Suite
}

func Test_DeploymentExposure(t *testing.T) {
	suite.Run(t, new(DeploymentExposureSuite))
}

var _ suite.SetupAllSuite = &DeploymentExposureSuite{}
var _ suite.TearDownTestSuite = &DeploymentExposureSuite{}

func (s *DeploymentExposureSuite) TearDownTest() {
	// Clear any messages received in fake central during the test run
	s.testContext.GetFakeCentral().ClearReceivedBuffer()
}

func (s *DeploymentExposureSuite) SetupSuite() {
	if testContext, err := resource.NewContext(s.T()); err != nil {
		s.Fail("failed to setup test context: %s", err)
	} else {
		s.testContext = testContext
	}
}

func (s *DeploymentExposureSuite) Test_ClusterIpPermutation() {
	s.testContext.RunWithResourcesPermutation(
		[]resource.YamlTestFile{
			NginxDeployment,
			NginxServiceClusterIP,
		}, "Cluster IP", func(t *testing.T, testC *resource.TestContext, _ map[string]k8s.Object) {
			// Test context already takes care of creating and destroying resources
			time.Sleep(2 * time.Second)
			assertLastDeploymentHasPortExposure(
				t,
				testC.GetFakeCentral().GetAllMessages(),
				[]*storage.PortConfig{
					{
						Protocol:      "TCP",
						ContainerPort: 9376,
						Exposure:      storage.PortConfig_INTERNAL,
						ExposureInfos: []*storage.PortConfig_ExposureInfo{
							{
								ServiceName: "nginx-svc-cluster-ip",
								ServicePort: 80,
								Level:       storage.PortConfig_INTERNAL,
							},
						},
					},
				},
				[]*storage.Alert{},
			)
			testC.GetFakeCentral().ClearReceivedBuffer()
		},
	)
}

func (s *DeploymentExposureSuite) Test_NodePortPermutation() {
	s.testContext.RunWithResourcesPermutation(
		[]resource.YamlTestFile{
			NginxDeployment,
			NginxServiceNodePort,
		}, "NodePort", func(t *testing.T, testC *resource.TestContext, _ map[string]k8s.Object) {
			// Test context already takes care of creating and destroying resources
			time.Sleep(2 * time.Second)
			assertLastDeploymentHasPortExposure(
				t,
				testC.GetFakeCentral().GetAllMessages(),
				[]*storage.PortConfig{
					{
						Protocol:      "TCP",
						ContainerPort: 80,
						Exposure:      storage.PortConfig_NODE,
						ExposureInfos: []*storage.PortConfig_ExposureInfo{
							{
								ServiceName: "nginx-svc-node-port",
								ServicePort: 80,
								NodePort:    30007,
								Level:       storage.PortConfig_NODE,
							},
						},
					},
				},
				[]*storage.Alert{
					{
						Policy: &storage.Policy{
							Name: "test-service",
						},
						State: storage.ViolationState_ACTIVE,
					},
				},
			)
			testC.GetFakeCentral().ClearReceivedBuffer()
		},
	)
}

func (s *DeploymentExposureSuite) Test_LoadBalancerPermutation() {
	s.testContext.RunWithResourcesPermutation(
		[]resource.YamlTestFile{
			NginxDeployment,
			NginxServiceLoadBalancer,
		}, "LoadBalancer", func(t *testing.T, testC *resource.TestContext, _ map[string]k8s.Object) {
			// Test context already takes care of creating and destroying resources
			time.Sleep(2 * time.Second)
			assertLastDeploymentHasPortExposure(
				t,
				testC.GetFakeCentral().GetAllMessages(),
				[]*storage.PortConfig{
					{
						Protocol:      "TCP",
						ContainerPort: 80,
						Exposure:      storage.PortConfig_EXTERNAL,
						ExposureInfos: []*storage.PortConfig_ExposureInfo{
							{
								ServiceName: "nginx-svc-load-balancer",
								ServicePort: 80,
								NodePort:    30007,
								Level:       storage.PortConfig_EXTERNAL,
							},
						},
					},
				},
				[]*storage.Alert{
					{
						Policy: &storage.Policy{
							Name: "test-service",
						},
						State: storage.ViolationState_ACTIVE,
					},
				},
			)
			testC.GetFakeCentral().ClearReceivedBuffer()
		},
	)
}

func (s *DeploymentExposureSuite) Test_NoExposure() {
	s.testContext.RunWithResources(
		[]resource.YamlTestFile{
			NginxDeployment,
		}, func(t *testing.T, testC *resource.TestContext, _ map[string]k8s.Object) {
			// Test context already takes care of creating and destroying resources
			time.Sleep(2 * time.Second)
			assertLastDeploymentHasPortExposure(
				t,
				testC.GetFakeCentral().GetAllMessages(),
				[]*storage.PortConfig{
					{
						Protocol:      "TCP",
						ContainerPort: 80,
						Exposure:      0,
					},
				},
				[]*storage.Alert{},
			)
			testC.GetFakeCentral().ClearReceivedBuffer()
		})
}

func (s *DeploymentExposureSuite) Test_MultipleDeploymentUpdates() {
	s.testContext.RunBare("Update permission level", func(t *testing.T, testC *resource.TestContext, _ map[string]k8s.Object) {
		deleteDep, err := testC.ApplyFileNoObject(context.Background(), "sensor-integration", NginxDeployment)
		defer utils.IgnoreError(deleteDep)
		require.NoError(t, err)

		deleteService, err := testC.ApplyFileNoObject(context.Background(), "sensor-integration", NginxServiceNodePort)
		defer utils.IgnoreError(deleteService)
		require.NoError(t, err)

		// Wait because of re-sync
		time.Sleep(3 * time.Second)

		assertLastDeploymentHasPortExposure(
			t,
			testC.GetFakeCentral().GetAllMessages(),
			[]*storage.PortConfig{
				{
					Protocol:      "TCP",
					ContainerPort: 80,
					Exposure:      storage.PortConfig_NODE,
					ExposureInfos: []*storage.PortConfig_ExposureInfo{
						{
							ServiceName: "nginx-svc-node-port",
							ServicePort: 80,
							NodePort:    30007,
							Level:       storage.PortConfig_NODE,
						},
					},
				},
			},
			[]*storage.Alert{
				{
					Policy: &storage.Policy{
						Name: "test-service",
					},
					State: storage.ViolationState_ACTIVE,
				},
			},
		)
		testC.GetFakeCentral().ClearReceivedBuffer()

		utils.IgnoreError(deleteService)

		// Wait because of re-sync
		time.Sleep(3 * time.Second)

		assertLastDeploymentHasPortExposure(
			t,
			testC.GetFakeCentral().GetAllMessages(),
			[]*storage.PortConfig{
				{
					Protocol:      "TCP",
					ContainerPort: 80,
					Exposure:      0,
				},
			},
			[]*storage.Alert{},
		)
		testC.GetFakeCentral().ClearReceivedBuffer()
	})
}

func (s *DeploymentExposureSuite) Test_NodePortPermutationWithPod() {
	s.testContext.RunWithResourcesPermutation(
		[]resource.YamlTestFile{
			NginxDeployment,
			NginxServiceNodePort,
		}, "NodePort", func(t *testing.T, testC *resource.TestContext, _ map[string]k8s.Object) {
			// Test context already takes care of creating and destroying resources
			time.Sleep(2 * time.Second)
			assertLastDeploymentHasPortExposure(
				t,
				testC.GetFakeCentral().GetAllMessages(),
				[]*storage.PortConfig{
					{
						Protocol:      "TCP",
						ContainerPort: 80,
						Exposure:      storage.PortConfig_NODE,
						ExposureInfos: []*storage.PortConfig_ExposureInfo{
							{
								ServiceName: "nginx-svc-node-port",
								ServicePort: 80,
								NodePort:    30007,
								Level:       storage.PortConfig_NODE,
							},
						},
					},
				},
				[]*storage.Alert{
					{
						Policy: &storage.Policy{
							Name: "test-service",
						},
						State: storage.ViolationState_ACTIVE,
					},
				},
			)
			testC.GetFakeCentral().ClearReceivedBuffer()
		},
	)
}
