package networkpolicy

import (
	"strconv"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
)

func TestNetworkPolicy(t *testing.T) {
	suite.Run(t, new(NetworkPolicySuite))
}

type NetworkPolicySuite struct {
	suite.Suite
}

func policy(classificationEnums []storage.NetworkPolicyType) *storage.NetworkPolicy {
	netpol := new(storage.NetworkPolicy)
	netpol.Spec = new(storage.NetworkPolicySpec)
	netpol.Spec.PolicyTypes = classificationEnums
	return netpol
}

func (suite *NetworkPolicySuite) Test_FilterForDeployment() {
	cases := map[string]struct {
		deploymentLabels        map[string]string
		specPodTemplateLabels   map[string]string
		netpolSelectors         []map[string]string
		expectedPoliciesMatched int
	}{
		"Match: one NW policy with one label": {
			deploymentLabels:      map[string]string{"app": "central"},
			specPodTemplateLabels: map[string]string{"app": "central"},
			netpolSelectors: []map[string]string{
				{"app": "central"},
			},
			expectedPoliciesMatched: 1,
		},
		"Match: one NW policy with two labels": {
			deploymentLabels:      map[string]string{"app": "central", "env": "prod"},
			specPodTemplateLabels: map[string]string{"app": "central", "env": "prod"},
			netpolSelectors: []map[string]string{
				{"app": "central", "env": "prod"},
			},
			expectedPoliciesMatched: 1,
		},
		"One Match: two NW policies with one label": {
			deploymentLabels:      map[string]string{"app": "central", "env": "prod"},
			specPodTemplateLabels: map[string]string{"app": "central", "env": "prod"},
			netpolSelectors: []map[string]string{
				{"app": "central"},
				{"app": "sensor"},
			},
			expectedPoliciesMatched: 1,
		},
		"Two Matches: two NW policies with one label": {
			deploymentLabels:      map[string]string{"app": "central", "env": "prod"},
			specPodTemplateLabels: map[string]string{"app": "central", "env": "prod"},
			netpolSelectors: []map[string]string{
				{"app": "central"},
				{"env": "prod"},
			},
			expectedPoliciesMatched: 2,
		},
		"No Match: NW policies shall not match against the deployment label even if 0 pods match": {
			deploymentLabels:      map[string]string{"app": "central-deployment", "env": "prod"},
			specPodTemplateLabels: map[string]string{"app": "central-pod", "env": "prod"},
			netpolSelectors: []map[string]string{
				{"app": "central-deployment", "env": "prod"},
			},
			expectedPoliciesMatched: 0,
		},
		"No Match: NW policies shall ignore the deployment labels even if pods have 0 labels": {
			deploymentLabels:      map[string]string{"app": "central", "env": "prod"},
			specPodTemplateLabels: map[string]string{},
			netpolSelectors: []map[string]string{
				{"app": "central"},
				{"env": "prod"},
			},
			expectedPoliciesMatched: 0,
		},
		"One Match: NW policies shall match against the pod labels only": {
			deploymentLabels:      map[string]string{"app": "central-deployment"},
			specPodTemplateLabels: map[string]string{"app": "central-pod"},
			netpolSelectors: []map[string]string{
				{"app": "central-pod"},
			},
			expectedPoliciesMatched: 1,
		},
		"No Match: one NW with two labels": {
			deploymentLabels:      map[string]string{"app": "central", "env": "prod"},
			specPodTemplateLabels: map[string]string{"app": "central", "env": "prod"},
			netpolSelectors: []map[string]string{
				{"app": "central", "env": "dev"},
			},
			expectedPoliciesMatched: 0,
		},
		"No Match: one NW policy with different labels": {
			deploymentLabels:      map[string]string{"app": "central"},
			specPodTemplateLabels: map[string]string{"app": "central"},
			netpolSelectors: []map[string]string{
				{"app": "sensor"},
			},
			expectedPoliciesMatched: 0,
		},
	}

	for name, testCase := range cases {
		suite.Run(name, func() {
			var policies []*storage.NetworkPolicy
			for idx, sel := range testCase.netpolSelectors {
				p := policy([]storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE})
				p.Spec.PodSelector = &storage.LabelSelector{MatchLabels: sel}
				p.Id = strconv.Itoa(idx)
				policies = append(policies, p)
			}

			dep := &storage.Deployment{
				Labels:    testCase.deploymentLabels,
				PodLabels: testCase.specPodTemplateLabels,
			}
			suite.Len(FilterForDeployment(policies, dep), testCase.expectedPoliciesMatched)
		})
	}
}

func (suite *NetworkPolicySuite) Test_GetNetworkPoliciesApplied() {
	cases := map[string]struct {
		policiesInStore map[string]*storage.NetworkPolicy
		hasIngres       bool
		hasEgress       bool
	}{
		"No policies for deployment": {
			policiesInStore: map[string]*storage.NetworkPolicy{},
			hasIngres:       false,
			hasEgress:       false,
		},
		"Ingress Policy": {
			policiesInStore: map[string]*storage.NetworkPolicy{
				"id1": policy([]storage.NetworkPolicyType{
					storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE,
				}),
			},
			hasIngres: true,
			hasEgress: false,
		},
		"Egress Policy": {
			policiesInStore: map[string]*storage.NetworkPolicy{
				"id1": policy([]storage.NetworkPolicyType{
					storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE,
				}),
			},
			hasIngres: false,
			hasEgress: true,
		},
		"Ingress and Egress on same policy object": {
			policiesInStore: map[string]*storage.NetworkPolicy{
				"id1": policy([]storage.NetworkPolicyType{
					storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE,
					storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE,
				}),
			},
			hasIngres: true,
			hasEgress: true,
		},
		"Ingress and Egress on different policy objects": {
			policiesInStore: map[string]*storage.NetworkPolicy{
				"id1": policy([]storage.NetworkPolicyType{
					storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE,
				}),
				"id2": policy([]storage.NetworkPolicyType{
					storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE,
				}),
			},
			hasIngres: true,
			hasEgress: true,
		},
		"Both missing if policy is UNSET": {
			policiesInStore: map[string]*storage.NetworkPolicy{
				"id1": policy([]storage.NetworkPolicyType{
					storage.NetworkPolicyType_UNSET_NETWORK_POLICY_TYPE,
				}),
			},
			hasIngres: false,
			hasEgress: false,
		},
	}

	for name, testCase := range cases {
		suite.Run(name, func() {
			aug := GenerateNetworkPoliciesAppliedObj(testCase.policiesInStore)
			suite.Equal(testCase.hasIngres, aug.HasIngressNetworkPolicy)
			suite.Equal(testCase.hasEgress, aug.HasEgressNetworkPolicy)
			suite.Len(aug.Policies, len(testCase.policiesInStore))
		})
	}
}
