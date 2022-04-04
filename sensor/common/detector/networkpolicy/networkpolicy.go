package networkpolicy

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/sensor/common/store"
)

type Finder struct {
	store store.NetworkPolicyStore
}

func NewFinder(store store.NetworkPolicyStore) *Finder {
	return &Finder{store: store}
}

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
