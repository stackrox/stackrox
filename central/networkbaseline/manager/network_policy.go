package manager

import (
	"encoding/json"
	"hash/fnv"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

// NetworkPolicyUpdateInfo captures the relevant fields from a storage.NetworkPolicy.
type NetworkPolicyUpdateInfo struct {
	Action            central.ResourceAction              `json:"action"`
	Namespace         string                              `json:"namespace"`
	PolicyPodSelector *storage.LabelSelector              `json:"policy_pod_selector"`
	IngressRule       []*storage.NetworkPolicyIngressRule `json:"ingress_rule"`
	EgressRule        []*storage.NetworkPolicyEgressRule  `json:"egress_rule"`
}

func (m *manager) fromNetworkPolicyProto(
	action central.ResourceAction,
	policy *storage.NetworkPolicy,
) NetworkPolicyUpdateInfo {
	return NetworkPolicyUpdateInfo{
		Action:            action,
		Namespace:         policy.GetNamespace(),
		PolicyPodSelector: policy.GetSpec().GetPodSelector(),
		IngressRule:       policy.GetSpec().GetIngress(),
		EgressRule:        policy.GetSpec().GetEgress(),
	}
}

func (m *manager) getHashOfNetworkPolicyWithResourceAction(
	action central.ResourceAction,
	policy *storage.NetworkPolicy,
) (uint64, error) {
	info := m.fromNetworkPolicyProto(action, policy)
	infoStr, err := json.Marshal(info)
	if err != nil {
		return 0, err
	}
	hashFunc := fnv.New64()
	_, err = hashFunc.Write(infoStr)
	if err != nil {
		return 0, err
	}
	return hashFunc.Sum64(), nil
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
	hash, err := m.getHashOfNetworkPolicyWithResourceAction(action, policy)
	if err != nil {
		return false, 0, err
	}
	if m.seenNetworkPolicies.Contains(hash) {
		// We have processed this action policy pair before
		return true, 0, nil
	}
	return false, hash, nil
}
