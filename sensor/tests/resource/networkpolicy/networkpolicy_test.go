package networkpolicy

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/tests/resource"
	"github.com/stretchr/testify/suite"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

var (
	NginxDeployment = resource.K8sResourceInfo{Kind: "Deployment", YamlFile: "nginx.yaml"}
	NetpolAllow443  = resource.K8sResourceInfo{Kind: "NetworkPolicy", YamlFile: "netpol-allow-443.yaml"}
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
	if result == nil {
		return false
	}

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
	s.testContext.NewRun(
		resource.WithResources([]resource.K8sResourceInfo{
			NginxDeployment, NetpolAllow443,
		}),
		resource.WithTestCase(func(t *testing.T, testC *resource.TestContext, _ map[string]k8s.Object) {
			// There's a caveat to this test: the state HAS a violation at the beginning, but
			// it disappears once re-sync kicks-in and processes the relationship betwee the network policy
			// and this deployment. Therefore, this test passes as is, but the opposite assertion would also
			// pass. e.g. "check if there IS an alert", because there will be a state where the alert is there.
			testC.LastViolationState("nginx-deployment", func(result *central.AlertResults) error {
				if checkIfAlertsHaveViolation(result, ingressNetpolViolationName) {
					return errors.Errorf("violation found for deployment %s and violation name %s", result.GetSource().String(), ingressNetpolViolationName)
				}
				return nil
			}, "Should not have a violation")
		}),
	)
}

func (s *NetworkPolicySuite) Test_DeploymentShouldHaveViolation() {
	s.testContext.NewRun(
		resource.WithResources([]resource.K8sResourceInfo{
			NginxDeployment,
		}),
		resource.WithTestCase(func(t *testing.T, testC *resource.TestContext, _ map[string]k8s.Object) {
			testC.LastViolationState("nginx-deployment", func(result *central.AlertResults) error {
				if !checkIfAlertsHaveViolation(result, ingressNetpolViolationName) {
					return errors.Errorf("violation not found for deployment %s and violation name %s", result.GetSource().String(), ingressNetpolViolationName)
				}
				return nil
			}, "Should have a violation")
		}),
	)
}
