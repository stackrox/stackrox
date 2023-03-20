package deployment

import (
	"context"

	"github.com/pkg/errors"
	networkPolicyDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/labels"
)

type IsolationDetails struct {
	PolicyIDs       []string
	IngressIsolated bool
	EgressIsolated  bool
}

type LabeledResource interface {
	GetClusterId() string
	GetNamespace() string
	GetPodLabels() map[string]string
}

type Matcher interface {
	GetIsolationDetails(deploymentLabels LabeledResource) IsolationDetails
}

type ClusterNamespace struct {
	Cluster   string
	Namespace string
}

type selectablePolicy struct {
	PolicyID string
	Selector labels.CompiledSelector
	Types    []storage.NetworkPolicyType
}

type policyMatcherImpl struct {
	netpolMap map[ClusterNamespace][]selectablePolicy
}

func BuildMatcher(store networkPolicyDS.DataStore, namespaceFilter []ClusterNamespace) (Matcher, error) {
	netpolMap, err := buildNetworkPolicies(store, namespaceFilter)
	if err != nil {
		return nil, err
	}

	return &policyMatcherImpl{
		netpolMap: netpolMap,
	}, nil
}

func buildNetworkPolicies(store networkPolicyDS.DataStore, namespace []ClusterNamespace) (map[ClusterNamespace][]selectablePolicy, error) {
	ctx := context.Background()
	result := map[ClusterNamespace][]selectablePolicy{}
	for _, clusterNs := range namespace {
		policies, err := store.GetNetworkPolicies(ctx, clusterNs.Cluster, clusterNs.Namespace)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get policies for %v", clusterNs)
		}

		result[clusterNs] = []selectablePolicy{}
		for _, policy := range policies {
			selector, err := labels.CompileSelector(policy.GetSpec().GetPodSelector())
			if err != nil {
				return nil, errors.Wrapf(err, "failed to create selector for policy: %s", policy.GetId())
			}
			result[clusterNs] = append(result[clusterNs], selectablePolicy{
				PolicyID: policy.GetId(),
				Selector: selector,
				Types:    policy.GetSpec().GetPolicyTypes(),
			})
		}
	}
	return result, nil
}

func (m *policyMatcherImpl) GetIsolationDetails(resource LabeledResource) IsolationDetails {
	// Get Policies from the same cluster and namespace
	policies, ok := m.netpolMap[ClusterNamespace{
		Namespace: resource.GetNamespace(),
		Cluster:   resource.GetClusterId(),
	}]

	if !ok {
		// There are no policies registered for this Namespace/Cluster, stop here
		return IsolationDetails{
			PolicyIDs:       nil,
			IngressIsolated: false,
			EgressIsolated:  false,
		}
	}

	isolationDetails := IsolationDetails{}
	for _, policy := range policies {
		if policy.Selector.MatchesAll() || policy.Selector.Matches(resource.GetPodLabels()) {
			isolationDetails.PolicyIDs = append(isolationDetails.PolicyIDs, policy.PolicyID)
			isolationDetails.IngressIsolated = isolationDetails.IngressIsolated || hasIngress(policy.Types)
			isolationDetails.EgressIsolated = isolationDetails.EgressIsolated || hasEgress(policy.Types)
		}
	}

	return isolationDetails
}

func hasEgress(types []storage.NetworkPolicyType) bool {
	return hasPolicyType(types, storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE)
}

func hasIngress(types []storage.NetworkPolicyType) bool {
	if len(types) == 0 {
		return true
	}
	return hasPolicyType(types, storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE)
}

func hasPolicyType(types []storage.NetworkPolicyType, t storage.NetworkPolicyType) bool {
	for _, pType := range types {
		if pType == t {
			return true
		}
	}
	return false
}
