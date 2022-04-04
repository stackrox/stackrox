package networkpolicy

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/sensor/common/store"
)

// Finder wraps store.NetworkPolicyStore and provides a convenient method to create
// augmentedobj.NetworkPoliciesApplied object from a deployment.
type Finder struct {
	store store.NetworkPolicyStore
}

// NewFinder creates a new instance of Finder with provided store.
func NewFinder(store store.NetworkPolicyStore) *Finder {
	return &Finder{store: store}
}

// GetNetworkPoliciesApplied creates an augmentedobj.NetworkPoliciesApplied object
// based on the provided deployment object and the in-memory store.
// Finder will use storage.NetworkPolicyType array property on storage.NetworkPolicy proto
// in order to determine presence or absence of a particular network policy type.
func (np *Finder) GetNetworkPoliciesApplied(deployment *storage.Deployment) *augmentedobjs.NetworkPoliciesApplied {
	var hasIngress, hasEgress bool
	for _, policy := range np.store.Find(deployment.Namespace, deployment.Labels) {
		for _, policyType := range policy.GetSpec().GetPolicyTypes() {
			if policyType == storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE {
				hasIngress = true
			} else if policyType == storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE {
				hasEgress = true
			}
		}
	}

	return &augmentedobjs.NetworkPoliciesApplied{
		MissingIngressNetworkPolicy: !hasIngress,
		MissingEgressNetworkPolicy:  !hasEgress,
	}
}
