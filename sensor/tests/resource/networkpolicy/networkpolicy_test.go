package networkpolicy

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/tests/resource"
	"github.com/stretchr/testify/suite"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

var (
	NginxDeployment = resource.YamlTestFile{Kind: "Deployment", File: "nginx.yaml"}
	NetpolAllow443  = resource.YamlTestFile{Kind: "NetworkPolicy", File: "netpol-allow-443.yaml"}
)

type NetworkPolicySuite struct {
	testContext *resource.TestContext
	suite.Suite
}

func Test_NetworkPolicy(t *testing.T) {
	suite.Run(t, new(NetworkPolicySuite))
}

var _ suite.SetupAllSuite = &NetworkPolicySuite{}
var _ suite.TearDownTestSuite = &NetworkPolicySuite{}

func (s *NetworkPolicySuite) SetupSuite() {
	if testContext, err := resource.NewContext(s.T()); err != nil {
		s.Fail("failed to setup test context: %s", err)
	} else {
		s.testContext = testContext
	}
}

func (s *NetworkPolicySuite) TearDownTest() {
	s.testContext.GetFakeCentral().ClearReceivedBuffer()
}

var (
	ingressNetpolViolationName = "Deployments should have at least one ingress Network Policy"
)

func checkIfAlertsHaveViolation(result *central.AlertResults, name string) bool {
	alerts := result.GetAlerts()
	if len(alerts) == 0 {
		return false
	}
	for _, alert := range result.GetAlerts() {
		if alert.GetPolicy().GetName() == name {
			return true
		}
	}
	return false
}

func (s *NetworkPolicySuite) Test_DeploymentShouldNotHaveViolation() {
	s.testContext.RunWithResources([]resource.YamlTestFile{
		NginxDeployment, NetpolAllow443,
	}, func(t *testing.T, testC *resource.TestContext, _ map[string]k8s.Object) {
		testC.LastViolationState("nginx-deployment", func(result *central.AlertResults) bool {
			return !checkIfAlertsHaveViolation(result, ingressNetpolViolationName)
		}, "Should not have a violation")
	})
}

func (s *NetworkPolicySuite) Test_DeploymentShouldHaveViolation() {
	s.testContext.RunWithResources([]resource.YamlTestFile{
		NginxDeployment,
	}, func(t *testing.T, testC *resource.TestContext, _ map[string]k8s.Object) {
		testC.LastViolationState("nginx-deployment", func(result *central.AlertResults) bool {
			return checkIfAlertsHaveViolation(result, ingressNetpolViolationName)
		}, "Should have a violation")
	})
}
