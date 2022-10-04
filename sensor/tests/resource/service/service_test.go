package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/tests/resource"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

const (
	nginxDeploymentName string = "nginx-deployment"
	nginxPodName        string = "nginx-rogue"
	servicePolicyName   string = "test-service"
)

var (
	NginxDeployment          = resource.YamlTestFile{Kind: "Deployment", File: "nginx.yaml"}
	NginxPod                 = resource.YamlTestFile{Kind: "Pod", File: "nginx-pod.yaml"}
	NginxServiceClusterIP    = resource.YamlTestFile{Kind: "Service", File: "nginx-service-cluster-ip.yaml"}
	NginxServiceNodePort     = resource.YamlTestFile{Kind: "Service", File: "nginx-service-node-port.yaml"}
	NginxServiceLoadBalancer = resource.YamlTestFile{Kind: "Service", File: "nginx-service-load-balancer.yaml"}
)

func checkAlert(alert *storage.Alert, result *central.AlertResults) error {
	for _, actualAlert := range result.GetAlerts() {
		if alert.GetPolicy().GetName() == actualAlert.GetPolicy().GetName() &&
			alert.GetState() == actualAlert.GetState() {
			return nil
		}
	}
	return errors.Errorf("Alert '%s' was not found", alert.GetPolicy().GetName())
}

func assertAlertTriggered(alert *storage.Alert) resource.AlertAssertFunc {
	return func(results *central.AlertResults) error {
		return checkAlert(alert, results)
	}
}

func assertAlertNotTriggered(alert *storage.Alert) resource.AlertAssertFunc {
	return func(results *central.AlertResults) error {
		if err := checkAlert(alert, results); err != nil {
			return nil
		}
		return errors.Errorf("alert '%s' should not be triggered", alert.GetPolicy().GetName())
	}
}

func checkPortConfig(deployment *storage.Deployment, ports []*storage.PortConfig) error {
	for _, expectedPort := range ports {
		foundPortConfig := false
		for _, port := range deployment.GetPorts() {
			if expectedPort.GetProtocol() == port.GetProtocol() &&
				expectedPort.GetContainerPort() == port.GetContainerPort() &&
				expectedPort.GetExposure() == port.GetExposure() {
				if len(expectedPort.GetExposureInfos()) != len(port.GetExposureInfos()) {
					continue
				}
				for _, expectedPortInfo := range expectedPort.GetExposureInfos() {
					foundPortInfo := false
					for _, portInfo := range port.GetExposureInfos() {
						if expectedPortInfo.GetServiceName() == portInfo.GetServiceName() {
							if expectedPortInfo.GetNodePort() != portInfo.GetNodePort() {
								return errors.Errorf("expected NodePort '%d' actual NodePort '%d'", expectedPortInfo.GetNodePort(), portInfo.GetNodePort())
							}
							if expectedPortInfo.GetServicePort() != portInfo.GetServicePort() {
								return errors.Errorf("expected ServicePort '%d' actual ServicePort '%d'", expectedPortInfo.GetServicePort(), portInfo.GetServicePort())
							}
							if expectedPortInfo.GetLevel() != portInfo.GetLevel() {
								return errors.Errorf("expected Level '%d' actual Level '%d'", expectedPortInfo.GetLevel(), portInfo.GetLevel())
							}
							foundPortInfo = true
						}
					}
					if !foundPortInfo {
						return errors.Errorf("PortInfo '%v' not found", expectedPort)
					}
				}
				foundPortConfig = true
			}
		}
		if !foundPortConfig {
			return errors.Errorf("PortConfig '%v' not found", expectedPort)
		}
	}
	return nil
}

func assertLastDeploymentHasPortExposure(ports []*storage.PortConfig) resource.AssertFunc {
	return func(deployment *storage.Deployment) error {
		return checkPortConfig(deployment, ports)
	}
}

func assertLastDeploymentMissingPortExposure(ports []*storage.PortConfig) resource.AssertFunc {
	return func(deployment *storage.Deployment) error {
		if err := checkPortConfig(deployment, ports); err != nil {
			return nil
		}
		return errors.Errorf("PortConfig '%v' should not be present", ports)
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
			testC.LastDeploymentState(nginxDeploymentName,
				assertLastDeploymentHasPortExposure([]*storage.PortConfig{
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
				),
				"'PortConfig' for Cluster IP service test not found",
			)
			testC.LastViolationState(nginxDeploymentName,
				assertAlertNotTriggered(
					&storage.Alert{
						Policy: &storage.Policy{
							Name: servicePolicyName,
						},
						State: storage.ViolationState_ACTIVE,
					},
				),
				fmt.Sprintf("Alert '%s' should not be triggered", servicePolicyName))
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
			testC.LastDeploymentState(nginxDeploymentName,
				assertLastDeploymentHasPortExposure([]*storage.PortConfig{
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
				),
				"'PortConfig' for Node Port service test not found",
			)
			testC.LastViolationState(nginxDeploymentName,
				assertAlertTriggered(
					&storage.Alert{
						Policy: &storage.Policy{
							Name: servicePolicyName,
						},
						State: storage.ViolationState_ACTIVE,
					},
				),
				fmt.Sprintf("Alert '%s' should be triggered", servicePolicyName))
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
			testC.LastDeploymentState(nginxDeploymentName,
				assertLastDeploymentHasPortExposure([]*storage.PortConfig{
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
				),
				"'PortConfig' for Load Balancer service test not found",
			)
			testC.LastViolationState(nginxDeploymentName,
				assertAlertTriggered(
					&storage.Alert{
						Policy: &storage.Policy{
							Name: servicePolicyName,
						},
						State: storage.ViolationState_ACTIVE,
					},
				),
				fmt.Sprintf("Alert '%s' should be triggered", servicePolicyName))
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
			testC.LastDeploymentState(nginxDeploymentName,
				assertLastDeploymentHasPortExposure([]*storage.PortConfig{
					{
						Protocol:      "TCP",
						ContainerPort: 80,
						Exposure:      0,
					},
				},
				),
				"PortConfig",
			)
			testC.LastViolationState(nginxDeploymentName,
				assertAlertNotTriggered(
					&storage.Alert{
						Policy: &storage.Policy{
							Name: servicePolicyName,
						},
						State: storage.ViolationState_ACTIVE,
					},
				),
				fmt.Sprintf("Alert '%s' should not be triggered", servicePolicyName))
			testC.GetFakeCentral().ClearReceivedBuffer()
		})
}

func (s *DeploymentExposureSuite) Test_MultipleDeploymentUpdates() {
	s.testContext.RunBare("Update Port Exposure", func(t *testing.T, testC *resource.TestContext, _ map[string]k8s.Object) {
		deleteDep, err := testC.ApplyFileNoObject(context.Background(), resource.DefaultNamespace, NginxDeployment)
		defer utils.IgnoreError(deleteDep)
		require.NoError(t, err)

		deleteService, err := testC.ApplyFileNoObject(context.Background(), resource.DefaultNamespace, NginxServiceNodePort)
		defer utils.IgnoreError(deleteService)
		require.NoError(t, err)

		testC.LastDeploymentState(nginxDeploymentName,
			assertLastDeploymentHasPortExposure([]*storage.PortConfig{
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
			),
			"'PortConfig' for Multiple Deployment Updates test not found",
		)
		testC.LastViolationState(nginxDeploymentName,
			assertAlertTriggered(
				&storage.Alert{
					Policy: &storage.Policy{
						Name: servicePolicyName,
					},
					State: storage.ViolationState_ACTIVE,
				},
			),
			fmt.Sprintf("Alert '%s' should be triggered", servicePolicyName))
		testC.GetFakeCentral().ClearReceivedBuffer()

		utils.IgnoreError(deleteService)

		testC.LastDeploymentState(nginxDeploymentName,
			assertLastDeploymentMissingPortExposure([]*storage.PortConfig{
				//{
				//	Protocol:      "TCP",
				//	ContainerPort: 80,
				//	Exposure:      0,
				// },
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
			),
			"'PortConfig' for Multiple Deployment Updates test found",
		)
		testC.LastViolationState(nginxDeploymentName,
			assertAlertNotTriggered(
				&storage.Alert{
					Policy: &storage.Policy{
						Name: servicePolicyName,
					},
					State: storage.ViolationState_RESOLVED,
				},
			),
			fmt.Sprintf("Alert '%s' should not be triggered", servicePolicyName))
		testC.GetFakeCentral().ClearReceivedBuffer()
	})
}

func (s *DeploymentExposureSuite) Test_NodePortPermutationWithPod() {
	s.testContext.RunWithResourcesPermutation(
		[]resource.YamlTestFile{
			NginxPod,
			NginxServiceNodePort,
		}, "PodNodePort", func(t *testing.T, testC *resource.TestContext, _ map[string]k8s.Object) {
			// Test context already takes care of creating and destroying resources
			testC.LastDeploymentState(nginxPodName,
				assertLastDeploymentHasPortExposure([]*storage.PortConfig{
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
				),
				"'PortConfig' for Node Port for Pod test not found",
			)
			testC.LastViolationState(nginxPodName,
				assertAlertTriggered(
					&storage.Alert{
						Policy: &storage.Policy{
							Name: servicePolicyName,
						},
						State: storage.ViolationState_ACTIVE,
					},
				),
				fmt.Sprintf("Alert '%s' should be triggered", servicePolicyName))
			testC.GetFakeCentral().ClearReceivedBuffer()
		},
	)
}
