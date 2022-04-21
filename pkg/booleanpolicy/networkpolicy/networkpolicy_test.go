package networkpolicy

import (
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

func (suite *NetworkPolicySuite) Test_GetNetworkPoliciesApplied() {
	cases := map[string]struct {
		policiesInStore map[string]*storage.NetworkPolicy
		missingIngress  bool
		missingEgress   bool
	}{
		"No policies for deployment": {
			policiesInStore: map[string]*storage.NetworkPolicy{},
			missingIngress:  true,
			missingEgress:   true,
		},
		"Ingress Policy": {
			policiesInStore: map[string]*storage.NetworkPolicy{
				"id1": policy([]storage.NetworkPolicyType{
					storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE,
				}),
			},
			missingIngress: false,
			missingEgress:  true,
		},
		"Egress Policy": {
			policiesInStore: map[string]*storage.NetworkPolicy{
				"id1": policy([]storage.NetworkPolicyType{
					storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE,
				}),
			},
			missingIngress: true,
			missingEgress:  false,
		},
		"Ingress and Egress on same policy object": {
			policiesInStore: map[string]*storage.NetworkPolicy{
				"id1": policy([]storage.NetworkPolicyType{
					storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE,
					storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE,
				}),
			},
			missingIngress: false,
			missingEgress:  false,
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
			missingIngress: false,
			missingEgress:  false,
		},
		"Both missing if policy is UNSET": {
			policiesInStore: map[string]*storage.NetworkPolicy{
				"id1": policy([]storage.NetworkPolicyType{
					storage.NetworkPolicyType_UNSET_NETWORK_POLICY_TYPE,
				}),
			},
			missingIngress: true,
			missingEgress:  true,
		},
	}

	for name, testCase := range cases {
		suite.Run(name, func() {
			aug := GetNetworkPoliciesApplied(testCase.policiesInStore)
			suite.Equal(testCase.missingIngress, aug.MissingIngressNetworkPolicy)
			suite.Equal(testCase.missingEgress, aug.MissingEgressNetworkPolicy)
			suite.Len(aug.Policies, len(testCase.policiesInStore))
		})
	}
}
