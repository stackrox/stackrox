package resources

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/labels"
	"github.com/stackrox/rox/pkg/reconcile"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/deduper"
	"github.com/stackrox/rox/sensor/common/detector/metrics"
	"github.com/stackrox/rox/sensor/common/store"

	"github.com/stackrox/rox/generated/storage"
)

/* Matching labels using selectors

Selector is a set of labels (map[string]string) that is used for matching the labels.
Labels are also represented as set (map[string]string) but they are passive in the matching process.
Note that selectors.Match(labels) != labels.Match(selectors)

"Matching objects must satisfy all the specified label constraints (here called selector terms), though they may have additional labels as well."
https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#equality-based-requirement

Selectors match labels if all selectors terms are contained in the set of labels.
Speaking differently, the set of selector terms belongs to the power-set of labels (minus the empty set).
(See https://en.wikipedia.org/wiki/Power_set)

Selected observations for matching selectors with labels including:
1. Policy with no selectors (podSelector: {}) matches all deployments in the namespace.
2. Policy with a selector (podSelector: matchLabels: app: web}) matches pod with matching labels "app: web" but not a service with matching labels "app: web".
3. Pod with no labels matches only against policies with no selectors.

Example:
A policy with selectors L1=V1,L2=V2 (having two selector terms: L1=V1 and L2=V2)
would match a deployment having labels (L1=V1,L2=V2), or (L1=V1,L2=V2,L3=V3),
but not for (L1=V1).
A deployment with labels L1=V1,L2=V2 could be matched with policies having the following selectors (L1=V1), (L2=V2), (L1=V1,L2=V2),
but not (L1=V1,L3=V3).
*/

/*
networkPolicyStoreImpl stores a set of network policies as seen in the K8s API.

This store is optimized for quick searches* of network policies that match a given deployment.
NetworkPolicies use label-selectors to match labels.
The Find operation returns all NetworkPolicies that would match a given set of labels (within a namespace).
Example:
  policiesMatchingDeployment := store.Find("default"), map[string]string{"app": "nginx"})

See ADR-0002 "Design In-Memory Data Store for Network Policies in Sensor" for alternative implementations that were considered

## Complexities

Assumed frequency of operations on the store:
- Find - arbitrarily often, at least once per each deployment update,
- Upsert Delete - once per NetworkPolicy update arriving from K8s API,
- Get - occasionally

Notation used:
- N = number of elements (network policies) in the store (for a given namespace)
- K = number of labels in a deployment (passed to the Find function)
- L = number of label selector terms in a network policy

### Computation

- O(Find) = O(N*K) - iterating over all N policies in a given namespace and matching K labels
- Upsert - O(1)
- Delete - O(1)

### Memory

The store stores each network policy exactly once O(n).
The operations on the store may allocate additional memory temporarily.

- Find: O(1)
- Upsert: O(1)
- Delete: O(1)
*/

var _ store.NetworkPolicyStore = (*networkPolicyStoreImpl)(nil)

type networkPolicyStoreImpl struct {
	lock sync.RWMutex
	// data: namespace -> result (map: policyID -> policy object ref)
	data map[string]map[string]*storage.NetworkPolicy
}

// ReconcileDelete is called after Sensor reconnects with Central and receives its state hashes.
// Reconciliacion ensures that Sensor and Central have the same state by checking whether a given resource
// shall be deleted from Central.
func (n *networkPolicyStoreImpl) ReconcileDelete(resType, resID string, _ uint64) ([]reconcile.Resource, error) {
	if resType != deduper.TypeNetworkPolicy.String() {
		return nil, errors.Errorf("invalid resource type %v", resType)
	}

	p := n.Get(resID)
	if p != nil {
		return nil, nil
	}
	// Resource on Central but not on Sensor, send for deletion
	return []reconcile.Resource{
		&reconcilePair{
			resID:   resID,
			resType: resType,
		},
	}, nil
}

func newNetworkPoliciesStore() *networkPolicyStoreImpl {
	return &networkPolicyStoreImpl{
		data: make(map[string]map[string]*storage.NetworkPolicy),
	}
}

// Cleanup deletes all entries from store
func (n *networkPolicyStoreImpl) Cleanup() {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.data = make(map[string]map[string]*storage.NetworkPolicy)
}

// Size returns number of network policies in the store
func (n *networkPolicyStoreImpl) Size() int {
	n.lock.RLock()
	defer n.lock.RUnlock()
	size := 0
	for _, m := range n.data {
		size += len(m)
	}
	return size
}

// All returns all network policies from all namespaces
func (n *networkPolicyStoreImpl) All() map[string]*storage.NetworkPolicy {
	n.lock.RLock()
	defer n.lock.RUnlock()
	// copying map to ensure that the store contents will not be mutated from outside
	result := make(map[string]*storage.NetworkPolicy)
	for _, m := range n.data {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

func (n *networkPolicyStoreImpl) updateStateMetric() {
	for ns, m := range n.data {
		metrics.ObserveNetworkPolicyStoreState(ns, len(m))
	}
}

// Delete removes network policy from the store
func (n *networkPolicyStoreImpl) Delete(ID, ns string) {
	n.lock.Lock()
	defer n.lock.Unlock()
	defer n.updateStateMetric()
	if policies, nsFound := n.data[ns]; nsFound {
		if policy, policyFound := policies[ID]; policyFound {
			metrics.ObserveNetworkPolicyStoreEvent("delete", ns, len(policy.GetSpec().GetPodSelector().GetMatchLabels()))
		}
		delete(n.data[ns], ID)
		if len(n.data[ns]) == 0 {
			delete(n.data, ns)
		}
	}
}

// Upsert adds or updates network policy based on the namespace and ID
func (n *networkPolicyStoreImpl) Upsert(np *storage.NetworkPolicy) {
	n.lock.Lock()
	defer n.lock.Unlock()
	defer n.updateStateMetric()

	if _, nsFound := n.data[np.GetNamespace()]; !nsFound {
		n.data[np.GetNamespace()] = make(map[string]*storage.NetworkPolicy)
	}
	event := "add"
	if _, exists := n.data[np.GetNamespace()][np.GetId()]; exists {
		event = "update"
	}
	metrics.ObserveNetworkPolicyStoreEvent(event, np.GetNamespace(), len(np.GetSpec().GetPodSelector().GetMatchLabels()))
	n.data[np.GetNamespace()][np.GetId()] = np
}

// Get retrieves network policy for a given ID or nil if the policy cannot be found
func (n *networkPolicyStoreImpl) Get(id string) *storage.NetworkPolicy {
	n.lock.RLock()
	defer n.lock.RUnlock()

	for _, m := range n.data {
		if obj, found := m[id]; found {
			return obj
		}
	}
	return nil
}

// Find returns set of NetworkPolicies that match the deployment labels
func (n *networkPolicyStoreImpl) Find(namespace string, podLabels map[string]string) map[string]*storage.NetworkPolicy {
	n.lock.RLock()
	defer n.lock.RUnlock()

	results := make(map[string]*storage.NetworkPolicy)
	nsPolicies, nsFound := n.data[namespace]
	if !nsFound {
		return results
	}

	// Pod with 0 labels should only match policies with 0 selectors.
	// Apparently 'labels.MatchLabels' does not cover this corner case
	if len(podLabels) == 0 {
		for id, policy := range nsPolicies {
			if policy.GetSpec().GetPodSelector().Size() == 0 {
				results[id] = policy
			}
		}
		return results
	}

	for id, policy := range nsPolicies {
		if labels.MatchLabels(policy.GetSpec().GetPodSelector(), podLabels) {
			results[id] = policy
		}
	}
	return results
}

// OnNamespaceDeleted reacts to a namespace deletion, deleting all network policies in this namespace from the store.
func (n *networkPolicyStoreImpl) OnNamespaceDeleted(namespace string) {
	n.lock.Lock()
	defer n.lock.Unlock()
	defer n.updateStateMetric()

	netpols, found := n.data[namespace]
	if !found {
		return
	}

	for id := range netpols {
		delete(n.data[namespace], id)
	}
	delete(n.data, namespace)
}
