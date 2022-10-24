package service

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/tests/resource"
	"github.com/stackrox/rox/sensor/testutils"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

const (
	nginxDeploymentName    string = "nginx-deployment"
	nginxPodName           string = "nginx-rogue"
	servicePolicyName      string = "test-service-%d"
	serviceNodePortFmt     string = "nginx-service-node-port-%d.yaml"
	serviceLoadBalancerFmt string = "nginx-service-load-balancer-%d.yaml"
)

var (
	NginxDeployment       = resource.YamlTestFile{Kind: "Deployment", File: "nginx.yaml"}
	NginxPod              = resource.YamlTestFile{Kind: "Pod", File: "nginx-pod.yaml"}
	NginxServiceClusterIP = resource.YamlTestFile{Kind: "Service", File: "nginx-service-cluster-ip.yaml"}
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
						return errors.Errorf("PortInfo '%+v' not found", expectedPortInfo)
					}
				}
				foundPortConfig = true
			}
		}
		if !foundPortConfig {
			return errors.Errorf("PortConfig '%+v' not found", expectedPort)
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
		return errors.Errorf("PortConfig '%+v' should not be present", ports)
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
	policies, err := testutils.GetPoliciesFromFile("data/policies.json")
	if err != nil {
		log.Fatalln(err)
	}
	config := resource.CentralConfig{
		InitialSystemPolicies: policies,
	}
	if testContext, err := resource.NewContextWithConfig(s.T(), config); err != nil {
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

func testForNodePortService(testContext *resource.TestContext, resources []resource.YamlTestFile, serviceName string, port int32, deploymentName string) {
	policyName := fmt.Sprintf(servicePolicyName, port)
	testContext.RunWithResources(
		resources, func(t *testing.T, testC *resource.TestContext, _ map[string]k8s.Object) {
			// Test context already takes care of creating and destroying resources
			testC.LastDeploymentState(deploymentName,
				assertLastDeploymentHasPortExposure([]*storage.PortConfig{
					{
						Protocol:      "TCP",
						ContainerPort: 80,
						Exposure:      storage.PortConfig_NODE,
						ExposureInfos: []*storage.PortConfig_ExposureInfo{
							{
								ServiceName: serviceName,
								ServicePort: 80,
								NodePort:    port,
								Level:       storage.PortConfig_NODE,
							},
						},
					},
				},
				),
				"'PortConfig' for Node Port service test not found",
			)
			testC.LastViolationState(deploymentName,
				assertAlertTriggered(
					&storage.Alert{
						Policy: &storage.Policy{
							Name: policyName,
						},
						State: storage.ViolationState_ACTIVE,
					},
				),
				fmt.Sprintf("Alert '%s' should be triggered", policyName))
			testC.GetFakeCentral().ClearReceivedBuffer()
		})
}

func (s *DeploymentExposureSuite) Test_NodePortPermutation() {
	var port int32 = 30006
	serviceNameFmt := "nginx-svc-node-port-%d"
	nginxServiceNodePort := resource.YamlTestFile{
		Kind: "Service",
		File: fmt.Sprintf(serviceNodePortFmt, port),
	}

	// Create deployment first
	testForNodePortService(s.testContext, []resource.YamlTestFile{
		NginxDeployment,
		nginxServiceNodePort,
	}, fmt.Sprintf(serviceNameFmt, port), port, nginxDeploymentName)

	port = 30007
	nginxServiceNodePort.File = fmt.Sprintf(serviceNodePortFmt, port)

	// Create Service first
	testForNodePortService(s.testContext, []resource.YamlTestFile{
		nginxServiceNodePort,
		NginxDeployment,
	}, fmt.Sprintf(serviceNameFmt, port), port, nginxDeploymentName)
}

func testForLoadBalancerService(testContext *resource.TestContext, resources []resource.YamlTestFile, serviceName string, port int32) {
	policyName := fmt.Sprintf(servicePolicyName, port)
	testContext.RunWithResources(
		resources, func(t *testing.T, testC *resource.TestContext, _ map[string]k8s.Object) {
			// Test context already takes care of creating and destroying resources
			testC.LastDeploymentState(nginxDeploymentName,
				assertLastDeploymentHasPortExposure([]*storage.PortConfig{
					{
						Protocol:      "TCP",
						ContainerPort: 80,
						Exposure:      storage.PortConfig_EXTERNAL,
						ExposureInfos: []*storage.PortConfig_ExposureInfo{
							{
								ServiceName: serviceName,
								ServicePort: 80,
								NodePort:    port,
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
							Name: policyName,
						},
						State: storage.ViolationState_ACTIVE,
					},
				),
				fmt.Sprintf("Alert '%s' should be triggered", policyName))
			testC.GetFakeCentral().ClearReceivedBuffer()
		},
	)

}

func (s *DeploymentExposureSuite) Test_LoadBalancerPermutation() {
	var port int32 = 30011
	serviceNameFmt := "nginx-svc-load-balancer-%d"
	nginxServiceLoadBalancer := resource.YamlTestFile{
		Kind: "Service",
		File: fmt.Sprintf(serviceLoadBalancerFmt, port),
	}

	// Create deployment first
	testForLoadBalancerService(s.testContext, []resource.YamlTestFile{
		NginxDeployment,
		nginxServiceLoadBalancer,
	}, fmt.Sprintf(serviceNameFmt, port), port)

	port = 30012
	nginxServiceLoadBalancer.File = fmt.Sprintf(serviceLoadBalancerFmt, port)

	// Create Service first
	testForLoadBalancerService(s.testContext, []resource.YamlTestFile{
		nginxServiceLoadBalancer,
		NginxDeployment,
	}, fmt.Sprintf(serviceNameFmt, port), port)
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

		var port int32 = 30008
		nginxServiceNodePort := resource.YamlTestFile{
			Kind: "Service",
			File: fmt.Sprintf(serviceNodePortFmt, port),
		}

		deleteService, err := testC.ApplyFileNoObject(context.Background(), resource.DefaultNamespace, nginxServiceNodePort)
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
							ServiceName: fmt.Sprintf("nginx-svc-node-port-%d", port),
							ServicePort: 80,
							NodePort:    port,
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
						Name: fmt.Sprintf(servicePolicyName, port),
					},
					State: storage.ViolationState_ACTIVE,
				},
			),
			fmt.Sprintf("Alert '%s' should be triggered", servicePolicyName))
		testC.GetFakeCentral().ClearReceivedBuffer()

		utils.IgnoreError(deleteService)

		testC.LastDeploymentState(nginxDeploymentName,
			assertLastDeploymentMissingPortExposure([]*storage.PortConfig{
				{
					Protocol:      "TCP",
					ContainerPort: 80,
					Exposure:      storage.PortConfig_NODE,
					ExposureInfos: []*storage.PortConfig_ExposureInfo{
						{
							ServiceName: fmt.Sprintf("nginx-svc-node-port-%d", port),
							ServicePort: 80,
							NodePort:    port,
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
						Name: fmt.Sprintf(servicePolicyName, port),
					},
					State: storage.ViolationState_RESOLVED,
				},
			),
			fmt.Sprintf("Alert '%s' should not be triggered", servicePolicyName))
		testC.GetFakeCentral().ClearReceivedBuffer()
	})
}

func (s *DeploymentExposureSuite) Test_NodePortPermutationWithPod() {
	var port int32 = 30009
	serviceNameFmt := "nginx-svc-node-port-%d"
	nginxServiceNodePort := resource.YamlTestFile{
		Kind: "Service",
		File: fmt.Sprintf(serviceNodePortFmt, port),
	}

	// Create deployment first
	testForNodePortService(s.testContext, []resource.YamlTestFile{
		NginxPod,
		nginxServiceNodePort,
	}, fmt.Sprintf(serviceNameFmt, port), port, nginxPodName)

	port = 30010
	nginxServiceNodePort.File = fmt.Sprintf(serviceNodePortFmt, port)

	// Create Service first
	testForNodePortService(s.testContext, []resource.YamlTestFile{
		nginxServiceNodePort,
		NginxPod,
	}, fmt.Sprintf(serviceNameFmt, port), port, nginxPodName)
}
