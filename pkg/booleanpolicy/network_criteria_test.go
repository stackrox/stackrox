package booleanpolicy

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/violationmessages/printer"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestNetworkCriteria(t *testing.T) {
	t.Setenv(features.CVEFixTimestampCriteria.EnvVar(), "true")
	suite.Run(t, new(NetworkCriteriaTestSuite))
}

type NetworkCriteriaTestSuite struct {
	basePoliciesTestSuite
}

func networkBaselineMessage(t testing.TB, flow *augmentedobjs.NetworkFlowDetails) *storage.Alert_Violation {
	violation, err := printer.GenerateNetworkFlowViolation(flow)
	assert.Nil(t, err)
	return violation
}

func assertNetworkBaselineMessagesEqual(t testing.TB, this, that []*storage.Alert_Violation) {
	thisWithoutTime := make([]*storage.Alert_Violation, 0, len(this))
	thatWithoutTime := make([]*storage.Alert_Violation, 0, len(that))
	for _, violation := range this {
		cp := violation.CloneVT()
		cp.Time = nil
		thisWithoutTime = append(thisWithoutTime, cp)
	}
	for _, violation := range that {
		cp := violation.CloneVT()
		cp.Time = nil
		thatWithoutTime = append(thatWithoutTime, cp)
	}
	protoassert.ElementsMatch(t, thisWithoutTime, thatWithoutTime)
}

func (suite *NetworkCriteriaTestSuite) TestNetworkBaselinePolicy() {
	deployment := fixtures.GetDeployment().CloneVT()
	suite.addDepAndImages(deployment)

	// Create a policy for triggering flows that are not in baseline
	whitelistGroup := policyGroupWithSingleKeyValue(fieldnames.UnexpectedNetworkFlowDetected, "true", false)

	policy := policyWithGroups(storage.EventSource_DEPLOYMENT_EVENT, whitelistGroup)
	m, err := BuildDeploymentWithNetworkFlowMatcher(policy)
	suite.NoError(err)

	srcName, dstName, port, protocol := "deployment-name", "ext-source-name", 1, storage.L4Protocol_L4_PROTOCOL_TCP
	flow := &augmentedobjs.NetworkFlowDetails{
		SrcEntityName:        srcName,
		SrcEntityType:        storage.NetworkEntityInfo_DEPLOYMENT,
		DstEntityName:        dstName,
		DstEntityType:        storage.NetworkEntityInfo_DEPLOYMENT,
		DstPort:              uint32(port),
		L4Protocol:           protocol,
		NotInNetworkBaseline: true,
		LastSeenTimestamp:    time.Now(),
	}

	violations, err := m.MatchDeploymentWithNetworkFlowInfo(nil, enhancedDeployment(deployment, suite.getImagesForDeployment(deployment)), flow)
	suite.NoError(err)
	assertNetworkBaselineMessagesEqual(
		suite.T(),
		violations.AlertViolations,
		[]*storage.Alert_Violation{networkBaselineMessage(suite.T(), flow)})

	// And if the flow is in the baseline, no violations should exist
	flow.NotInNetworkBaseline = false
	violations, err = m.MatchDeploymentWithNetworkFlowInfo(nil, enhancedDeployment(deployment, suite.getImagesForDeployment(deployment)), flow)
	suite.NoError(err)
	suite.Empty(violations)
}

func (suite *NetworkCriteriaTestSuite) TestNetworkPolicyFields() {
	testCases := map[string]struct {
		netpolsApplied *augmentedobjs.NetworkPoliciesApplied
		alerts         []*storage.Alert_Violation
	}{
		"Missing Ingress Network Policy": {
			netpolsApplied: &augmentedobjs.NetworkPoliciesApplied{
				HasIngressNetworkPolicy: false,
				HasEgressNetworkPolicy:  true,
			},
			alerts: []*storage.Alert_Violation{
				{Message: "The deployment is missing Ingress Network Policy.", Type: storage.Alert_Violation_NETWORK_POLICY},
			},
		},
		"Missing Egress Network Policy": {
			netpolsApplied: &augmentedobjs.NetworkPoliciesApplied{
				HasIngressNetworkPolicy: true,
				HasEgressNetworkPolicy:  false,
			},
			alerts: []*storage.Alert_Violation{
				{Message: "The deployment is missing Egress Network Policy.", Type: storage.Alert_Violation_NETWORK_POLICY},
			},
		},
		"Both policies missing": {
			netpolsApplied: &augmentedobjs.NetworkPoliciesApplied{
				HasIngressNetworkPolicy: false,
				HasEgressNetworkPolicy:  false,
			},
			alerts: []*storage.Alert_Violation{
				{Message: "The deployment is missing Ingress Network Policy.", Type: storage.Alert_Violation_NETWORK_POLICY},
				{Message: "The deployment is missing Egress Network Policy.", Type: storage.Alert_Violation_NETWORK_POLICY},
			},
		},
		"No alerts": {
			netpolsApplied: &augmentedobjs.NetworkPoliciesApplied{
				HasIngressNetworkPolicy: true,
				HasEgressNetworkPolicy:  true,
			},
			alerts: []*storage.Alert_Violation(nil),
		},
		"No violations on nil augmentedobj": {
			netpolsApplied: nil,
			alerts:         []*storage.Alert_Violation(nil),
		},
		"Policies attached to augmentedobj": {
			netpolsApplied: &augmentedobjs.NetworkPoliciesApplied{
				HasIngressNetworkPolicy: false,
				HasEgressNetworkPolicy:  true,
				Policies: map[string]*storage.NetworkPolicy{
					"ID1": {Id: "ID1", Name: "policy1"},
				},
			},
			alerts: []*storage.Alert_Violation{
				{
					Message: "The deployment is missing Ingress Network Policy.",
					Type:    storage.Alert_Violation_NETWORK_POLICY,
					MessageAttributes: &storage.Alert_Violation_KeyValueAttrs_{
						KeyValueAttrs: &storage.Alert_Violation_KeyValueAttrs{
							Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{
								{Key: printer.PolicyID, Value: "ID1"},
								{Key: printer.PolicyName, Value: "policy1"},
							},
						},
					},
				},
			},
		},
	}

	for name, testCase := range testCases {
		suite.Run(name, func() {
			deployment := fixtures.GetDeployment().CloneVT()
			missingIngressPolicy := policyWithSingleKeyValue(fieldnames.HasIngressNetworkPolicy, "false", false)
			missingEgressPolicy := policyWithSingleKeyValue(fieldnames.HasEgressNetworkPolicy, "false", false)

			enhanced := enhancedDeploymentWithNetworkPolicies(
				deployment,
				suite.getImagesForDeployment(deployment),
				testCase.netpolsApplied,
			)

			v1 := suite.getViolations(missingIngressPolicy, enhanced)
			v2 := suite.getViolations(missingEgressPolicy, enhanced)

			allAlerts := append(v1.AlertViolations, v2.AlertViolations...)
			for i, expected := range testCase.alerts {
				suite.Equal(expected.GetType(), allAlerts[i].GetType())
				suite.Equal(expected.GetMessage(), allAlerts[i].GetMessage())
				protoassert.Equal(suite.T(), expected.GetKeyValueAttrs(), allAlerts[i].GetKeyValueAttrs())
				// We do not want to compare time, as the violation timestamp uses now()
				suite.NotNil(allAlerts[i].GetTime())
			}
		})
	}
}
