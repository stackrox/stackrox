package networkpolicy

import (
	"testing"
	"time"

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
	// Clear any messages received in fake central during the test run
	s.testContext.GetFakeCentral().ClearReceivedBuffer()
}

var (
	ingressNetpolViolationName = "Deployments should have at least one ingress Network Policy"
)

func (s *NetworkPolicySuite) Test_DeploymentShouldNotHaveViolation() {
	s.testContext.RunWithResources([]resource.YamlTestFile{
		NginxDeployment, NetpolAllow443,
	}, func(t *testing.T, testC *resource.TestContext, _ map[string]k8s.Object) {
		// Test context already takes care of creating and destroying resources
		time.Sleep(2 * time.Second)

		messages := testC.GetFakeCentral().GetAllMessages()

		alerts := getAllAlertsForDeploymentName(messages, "nginx-deployment")
		lastViolationState := alerts[len(alerts)-1]
		var hasViolation bool
		for _, alert := range lastViolationState.GetEvent().GetAlertResults().GetAlerts() {
			if alert.GetPolicy().GetName() == ingressNetpolViolationName {
				hasViolation = true
				break
			}
		}
		s.Require().False(hasViolation, "Should not have violation %s, but found in last violation state")
	})
}

func (s *NetworkPolicySuite) Test_DeploymentShouldHaveViolation() {
	s.testContext.RunWithResources([]resource.YamlTestFile{
		NginxDeployment,
	}, func(t *testing.T, testC *resource.TestContext, _ map[string]k8s.Object) {
		// Test context already takes care of creating and destroying resources
		time.Sleep(2 * time.Second)

		messages := testC.GetFakeCentral().GetAllMessages()

		alerts := getAllAlertsForDeploymentName(messages, "nginx-deployment")
		lastViolationState := alerts[len(alerts)-1]
		var hasViolation bool
		for _, alert := range lastViolationState.GetEvent().GetAlertResults().GetAlerts() {
			if alert.GetPolicy().GetName() == ingressNetpolViolationName {
				hasViolation = true
				break
			}
		}
		s.Require().True(hasViolation, "Should have violation %s, but not found in last violation state")
	})
}

// TODO:
// - Deployment updated removes the violation
// - Two deployments matching a network policy -> netpol update triggers two deployment updates
// - Network policy matching one deployment, updates selector to match another -> every deployment is updated

func getAllAlertsForDeploymentName(messages []*central.MsgFromSensor, name string) []*central.MsgFromSensor {
	var selected []*central.MsgFromSensor
	for _, m := range messages {
		for _, alert := range m.GetEvent().GetAlertResults().GetAlerts() {
			if alert.GetDeployment().GetName() == name {
				selected = append(selected, m)
				break
			}
		}
	}
	return selected
}
