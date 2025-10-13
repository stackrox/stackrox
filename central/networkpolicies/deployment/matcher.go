package deployment

import (
	"context"

	"github.com/pkg/errors"
	networkPolicyDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/labels"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
)

var (
	log = logging.LoggerForModule()
)

// IsolationDetails represents the isolation level of a deployment.
type IsolationDetails struct {
	PolicyIDs       []string
	IngressIsolated bool
	EgressIsolated  bool
}

// LabeledResource is likely network graph node that belongs to a cluster and a namespace, and should be
// selectable by Pod Selectors.
type LabeledResource interface {
	GetClusterId() string
	GetNamespace() string
	GetPodLabels() map[string]string
}

// Matcher interface for generating Isolation Details for deployments based on Network Policies matched.
type Matcher interface {
	GetIsolationDetails(deploymentLabels LabeledResource) IsolationDetails
}

// ClusterNamespace is a helper struct to index network policies based on their cluster and namespace.
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

// BuildMatcher creates a matcher with pre-loaded Network Policies from a set of ClusterNamespace filter.
func BuildMatcher(ctx context.Context, store networkPolicyDS.DataStore, namespaceFilter set.Set[ClusterNamespace]) (Matcher, error) {
	netpolMap, err := buildNetworkPolicies(ctx, store, namespaceFilter)
	if err != nil {
		return nil, err
	}

	return &policyMatcherImpl{
		netpolMap: netpolMap,
	}, nil
}

func buildNetworkPolicies(ctx context.Context, store networkPolicyDS.DataStore, namespace set.Set[ClusterNamespace]) (map[ClusterNamespace][]selectablePolicy, error) {
	result := map[ClusterNamespace][]selectablePolicy{}
	for clusterNs := range namespace {
		policies, err := store.GetNetworkPolicies(ctx, clusterNs.Cluster, clusterNs.Namespace)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get policies for %v", clusterNs)
		}

		for _, policy := range policies {
			selector, err := labels.CompileSelector(policy.GetSpec().GetPodSelector())
			if err != nil {
				log.Warnf("compile error in Network Policy (cluster:%s namespace:%s name:%s) for selector: %+v",
					policy.GetClusterName(),
					policy.GetNamespace(),
					policy.GetName(),
					policy.GetSpec().GetPodSelector())
				continue
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

// GetIsolationDetails will iterate over preloaded Network Policies and return the isolation level of
// the resource provided as parameter.
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
	return len(types) == 0 || hasPolicyType(types, storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE)
}

func hasPolicyType(types []storage.NetworkPolicyType, t storage.NetworkPolicyType) bool {
	for _, pType := range types {
		if pType == t {
			return true
		}
	}
	return false
}
