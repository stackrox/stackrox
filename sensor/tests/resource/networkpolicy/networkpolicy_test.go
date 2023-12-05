package networkpolicy

import (
	"context"
	"log"
	"testing"

	"github.com/stackrox/rox/sensor/tests/helper"
	"github.com/stackrox/rox/sensor/testutils"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

var (
	NginxDeployment            = helper.K8sResourceInfo{Kind: "Deployment", YamlFile: "nginx.yaml"}
	IngressPolicyAllow443      = helper.K8sResourceInfo{Kind: "NetworkPolicy", YamlFile: "netpol-allow-443.yaml"}
	EgressPolicyBlockAllEgress = helper.K8sResourceInfo{Kind: "NetworkPolicy", YamlFile: "netpol-block-egress.yaml"}
)

type NetworkPolicySuite struct {
	testContext *helper.TestContext
	suite.Suite
}

func Test_NetworkPolicy(t *testing.T) {
	suite.Run(t, new(NetworkPolicySuite))
}

var _ suite.SetupAllSuite = &NetworkPolicySuite{}
var _ suite.TearDownTestSuite = &NetworkPolicySuite{}

func (s *NetworkPolicySuite) SetupSuite() {
	policies, err := testutils.GetPoliciesFromFile("data/policies.json")
	if err != nil {
		log.Fatalln(err)
	}
	cfg := helper.DefaultCentralConfig()
	cfg.InitialSystemPolicies = policies

	if testContext, err := helper.NewContextWithConfig(s.T(), cfg); err != nil {
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
	egressNetpolViolationName  = "Deployments should have at least one egress Network Policy"
)

func (s *NetworkPolicySuite) Test_NetworkPolicyViolations() {
	s.testContext.RunTest(s.T(),
		helper.WithTestCase(func(t *testing.T, tc *helper.TestContext, objects map[string]k8s.Object) {
			ctx := context.Background()
			k8sDeployment := &appsv1.Deployment{}
			_, err := tc.ApplyResourceAndWait(ctx, t, helper.DefaultNamespace, &NginxDeployment, k8sDeployment, nil)
			require.NoError(t, err)

			deploymentID := string(k8sDeployment.GetUID())

			tc.LastViolationStateByID(t, deploymentID, helper.AssertViolationsMatch(egressNetpolViolationName, ingressNetpolViolationName), "", true)

			_, err = tc.ApplyResourceAndWaitNoObject(ctx, t, helper.DefaultNamespace, EgressPolicyBlockAllEgress, nil)
			require.NoError(t, err)

			tc.LastViolationStateByID(t, deploymentID, helper.AssertViolationsMatch(ingressNetpolViolationName), "", true)

			_, err = tc.ApplyResourceAndWaitNoObject(ctx, t, helper.DefaultNamespace, IngressPolicyAllow443, nil)
			require.NoError(t, err)

			tc.LastViolationStateByID(t, deploymentID, helper.AssertNoViolations(), "", true)
		}))
}
