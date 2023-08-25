package networkpolicy

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/labels"
)

// GenerateNetworkPoliciesAppliedObj creates an augmentedobj.NetworkPoliciesApplied object
// based on a provided map of network policies.
func GenerateNetworkPoliciesAppliedObj(networkPolicies map[string]*storage.NetworkPolicy) *augmentedobjs.NetworkPoliciesApplied {
	var hasIngress, hasEgress bool
	for _, policy := range networkPolicies {
		for _, policyType := range policy.GetSpec().GetPolicyTypes() {
			hasIngress = hasIngress || policyType == storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE
			hasEgress = hasEgress || policyType == storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE
			if hasIngress && hasEgress {
				return &augmentedobjs.NetworkPoliciesApplied{
					HasIngressNetworkPolicy: true,
					HasEgressNetworkPolicy:  true,
					Policies:                networkPolicies,
				}
			}
		}
	}

	return &augmentedobjs.NetworkPoliciesApplied{
		HasIngressNetworkPolicy: hasIngress,
		HasEgressNetworkPolicy:  hasEgress,
		Policies:                networkPolicies,
	}
}

// FilterForDeployment receives a deployment and slice of NetworkPolicies that represent the current state of Network
// Policies in a cluster. It returns a map filtered by only the Network Policies that match the given deployment.
func FilterForDeployment(networkPolicies []*storage.NetworkPolicy, deployment *storage.Deployment) map[string]*storage.NetworkPolicy {
	matchedNetworkPolicies := map[string]*storage.NetworkPolicy{}
	for _, p := range networkPolicies {
		if labels.MatchLabels(p.GetSpec().GetPodSelector(), deployment.GetPodLabels()) {
			matchedNetworkPolicies[p.GetId()] = p
		}
	}
	return matchedNetworkPolicies
}
