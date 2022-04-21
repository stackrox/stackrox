package networkpolicy

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
)

// GetNetworkPoliciesApplied creates an augmentedobj.NetworkPoliciesApplied object
// based on a provided map of network policies.
func GetNetworkPoliciesApplied(networkPolicies map[string]*storage.NetworkPolicy) *augmentedobjs.NetworkPoliciesApplied {
	var hasIngress, hasEgress bool
	for _, policy := range networkPolicies {
		for _, policyType := range policy.GetSpec().GetPolicyTypes() {
			hasIngress = hasIngress || policyType == storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE
			hasEgress = hasEgress || policyType == storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE
			if hasIngress && hasEgress {
				return &augmentedobjs.NetworkPoliciesApplied{
					MissingIngressNetworkPolicy: false,
					MissingEgressNetworkPolicy:  false,
					Policies:                    networkPolicies,
				}
			}
		}
	}

	return &augmentedobjs.NetworkPoliciesApplied{
		MissingIngressNetworkPolicy: !hasIngress,
		MissingEgressNetworkPolicy:  !hasEgress,
		Policies:                    networkPolicies,
	}
}
