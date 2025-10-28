package manager

import (
	"github.com/stackrox/hashstructure"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

func getHashOfNetworkPolicyWithResourceAction(
	action central.ResourceAction,
	policy *storage.NetworkPolicy,
) (uint64, error) {
	info := []any{
		action,
		policy.GetNamespace(),
		policy.GetSpec().GetPodSelector(),
		policy.GetSpec().GetIngress(),
		policy.GetSpec().GetEgress(),
	}
	return hashstructure.Hash(info, nil)
}

// Returns:
// - bool: should ignore this policy or not
// - uint64: if above is false, contains the hash of the action, policy pair
// The caller should update the state of manager to include this hash once the relevant
// processing has finished
func (m *manager) shouldIgnoreNetworkPolicy(
	action central.ResourceAction,
	policy *storage.NetworkPolicy,
) (bool, uint64, error) {
	hash, err := getHashOfNetworkPolicyWithResourceAction(action, policy)
	if err != nil {
		return false, 0, err
	}
	if m.seenNetworkPolicies.Contains(hash) {
		// We have processed this action policy pair before
		return true, 0, nil
	}
	return false, hash, nil
}
