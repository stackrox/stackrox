package generator

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/labels"
)

func checkPolicyType(policy *storage.NetworkPolicy, policyType storage.NetworkPolicyType) bool {
	for _, ty := range policy.GetSpec().GetPolicyTypes() {
		if ty == policyType {
			return true
		}
	}
	return false
}

func groupNetworkPolicies(policies []*storage.NetworkPolicy) (ingressPolicies, egressPolicies map[string][]*storage.NetworkPolicy) {
	ingressPolicies = make(map[string][]*storage.NetworkPolicy)
	egressPolicies = make(map[string][]*storage.NetworkPolicy)
	for _, policy := range policies {
		ns := policy.GetNamespace()
		if checkPolicyType(policy, storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE) {
			ingressPolicies[ns] = append(ingressPolicies[ns], policy)
		}
		if checkPolicyType(policy, storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE) {
			egressPolicies[ns] = append(egressPolicies[ns], policy)
		}
	}

	return
}

func hasMatchingPolicy(deployment *storage.Deployment, policies []*storage.NetworkPolicy) bool {
	for _, policy := range policies {
		if policy.GetNamespace() != deployment.GetNamespace() {
			continue
		}

		if labels.MatchLabels(policy.GetSpec().GetPodSelector(), deployment.GetPodLabels()) {
			return true
		}
	}
	return false
}
