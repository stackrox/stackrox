package resources

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/store"
)

var _ store.NetPoliciesStore = (*NetworkPolicyStore)(nil)

// NetworkPolicyStore stores a mapping of network policies names to their ids.
type NetworkPolicyStore struct {
	lock sync.RWMutex

	netPolicies         map[string]*storage.NetworkPolicy
	netPolicyNamesToIDs map[string]string
}

func newNetworkPolicyStore() *NetworkPolicyStore {
	return &NetworkPolicyStore{
		netPolicies:         make(map[string]*storage.NetworkPolicy),
		netPolicyNamesToIDs: make(map[string]string),
	}
}

// GetAll returns all Network Policies
func (n *NetworkPolicyStore) GetAll() []*storage.NetworkPolicy {
	rel := make([]*storage.NetworkPolicy, 0, len(n.netPolicies))
	for _, policy := range n.netPolicies {
		rel = append(rel, policy)
	}
	return rel
}

// Get retrieves Network Policy by ID
func (n *NetworkPolicyStore) Get(id string) *storage.NetworkPolicy {
	if policy, found := n.netPolicies[id]; found {
		return policy
	}
	return nil
}

func (n *NetworkPolicyStore) addNetPolicy(np *storage.NetworkPolicy) {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.netPolicies[np.GetId()] = np
	n.netPolicyNamesToIDs[np.GetName()] = np.GetId()
}

func (n *NetworkPolicyStore) deleteNetPolicy(np *storage.NetworkPolicy) {
	n.lock.Lock()
	defer n.lock.Unlock()

	delete(n.netPolicies, np.GetId())
	delete(n.netPolicyNamesToIDs, np.GetName())
}

func (n *NetworkPolicyStore) update(np *storage.NetworkPolicy) {
	n.lock.Lock()
	defer n.lock.Unlock()

	if updatedPolicy, found := n.netPolicies[np.GetId()]; found {
		n.netPolicies[updatedPolicy.GetId()] = np
		n.netPolicyNamesToIDs[updatedPolicy.GetName()] = np.GetId()
	} else {
		n.addNetPolicy(np)
	}
}
