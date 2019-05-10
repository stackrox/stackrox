package generator

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	dDS "github.com/stackrox/rox/central/deployment/datastore"
	nsDS "github.com/stackrox/rox/central/namespace/datastore"
	flowStore "github.com/stackrox/rox/central/networkflow/store"
	npDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

const (
	generatedNetworkPolicyLabel = `network-policy-generator.stackrox.io/generated`

	networkPolicyAPIVersion = `networking.k8s.io/v1`
)

func isGeneratedPolicy(policy *storage.NetworkPolicy) bool {
	_, ok := policy.GetLabels()[generatedNetworkPolicyLabel]
	return ok
}

type generator struct {
	networkPolicies npDS.DataStore
	deploymentStore dDS.DataStore
	namespacesStore nsDS.DataStore
	globalFlowStore flowStore.ClusterStore
}

func markGeneratedPoliciesForDeletion(policies []*storage.NetworkPolicy) ([]*storage.NetworkPolicy, []*storage.NetworkPolicyReference) {
	var userPolicies []*storage.NetworkPolicy
	var toDelete []*storage.NetworkPolicyReference

	for _, policy := range policies {
		if isGeneratedPolicy(policy) {
			toDelete = append(toDelete, &storage.NetworkPolicyReference{
				Name:      policy.GetName(),
				Namespace: policy.GetNamespace(),
			})
		} else {
			userPolicies = append(userPolicies, policy)
		}
	}

	return userPolicies, toDelete
}

func markAllPoliciesForDeletion(policies []*storage.NetworkPolicy) []*storage.NetworkPolicyReference {
	toDelete := make([]*storage.NetworkPolicyReference, 0, len(policies))
	for _, policy := range policies {
		toDelete = append(toDelete, &storage.NetworkPolicyReference{
			Name:      policy.GetName(),
			Namespace: policy.GetNamespace(),
		})
	}
	return toDelete
}

func (g *generator) getNetworkPolicies(ctx context.Context, deleteExistingMode v1.GenerateNetworkPoliciesRequest_DeleteExistingPoliciesMode, clusterID string) ([]*storage.NetworkPolicy, []*storage.NetworkPolicyReference, error) {
	policies, err := g.networkPolicies.GetNetworkPolicies(ctx, clusterID, "")
	if err != nil {
		return nil, nil, errors.Wrap(err, "obtaining network policies")
	}

	switch deleteExistingMode {
	case v1.GenerateNetworkPoliciesRequest_NONE:
		return policies, nil, nil
	case v1.GenerateNetworkPoliciesRequest_GENERATED_ONLY:
		userPolicies, toDelete := markGeneratedPoliciesForDeletion(policies)
		return userPolicies, toDelete, nil
	case v1.GenerateNetworkPoliciesRequest_ALL:
		return nil, markAllPoliciesForDeletion(policies), nil
	default:
		return nil, nil, fmt.Errorf("invalid 'delete existing' mode %v", deleteExistingMode)
	}
}

func (g *generator) generateGraph(ctx context.Context, clusterID string, since *types.Timestamp) (map[networkgraph.Entity]*node, error) {
	clusterFlowStore := g.globalFlowStore.GetFlowStore(clusterID)
	if clusterFlowStore == nil {
		return nil, fmt.Errorf("could not obtain flow store for cluster %q", clusterID)
	}

	allFlows, _, err := clusterFlowStore.GetAllFlows(since)
	if err != nil {
		return nil, errors.Wrapf(err, "could not obtain network flow information for cluster %q", clusterID)
	}

	deployments, err := g.deploymentStore.SearchRawDeployments(ctx, &v1.Query{
		Query: &v1.Query_BaseQuery{
			BaseQuery: &v1.BaseQuery{
				Query: &v1.BaseQuery_MatchFieldQuery{
					MatchFieldQuery: &v1.MatchFieldQuery{
						Field: "Cluster Id",
						Value: clusterID,
					},
				},
			},
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "could not obtain deployments for cluster %q", clusterID)
	}

	return buildGraph(deployments, allFlows), nil
}

func generatePolicy(node *node, namespacesByName map[string]*storage.NamespaceMetadata, ingressPolicies, egressPolicies map[string][]*storage.NetworkPolicy) *storage.NetworkPolicy {
	if hasMatchingPolicy(node.deployment, ingressPolicies[node.deployment.GetNamespace()]) {
		return nil
	}

	policy := &storage.NetworkPolicy{
		Name:        fmt.Sprintf("stackrox-generated-%s", node.deployment.GetName()),
		Namespace:   node.deployment.GetNamespace(),
		ClusterId:   node.deployment.GetClusterId(),
		ClusterName: node.deployment.GetClusterName(),
		Labels: map[string]string{
			generatedNetworkPolicyLabel: "true",
		},
		ApiVersion: networkPolicyAPIVersion,
		Spec: &storage.NetworkPolicySpec{
			PodSelector: labelSelectorForDeployment(node.deployment),
		},
	}

	ingressRule := generateIngressRule(node, namespacesByName)
	if ingressRule != nil {
		policy.Spec.Ingress = append(policy.Spec.Ingress, ingressRule)
	}
	policy.Spec.PolicyTypes = append(policy.Spec.PolicyTypes, storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE)

	return policy
}

func (g *generator) generatePolicies(graph map[networkgraph.Entity]*node, deploymentIDs set.StringSet, namespacesByName map[string]*storage.NamespaceMetadata, existingPolicies []*storage.NetworkPolicy) []*storage.NetworkPolicy {
	ingressPolicies, egressPolicies := groupNetworkPolicies(existingPolicies)

	var generatedPolicies []*storage.NetworkPolicy
	for _, node := range graph {
		if node.deployment == nil {
			continue
		}
		if isSystemDeployment(node.deployment) || (deploymentIDs.IsInitialized() && !deploymentIDs.Contains(node.deployment.GetId())) {
			continue
		}

		policy := generatePolicy(node, namespacesByName, ingressPolicies, egressPolicies)
		if policy != nil {
			generatedPolicies = append(generatedPolicies, policy)
		}
	}

	return generatedPolicies
}

func (g *generator) Generate(ctx context.Context, req *v1.GenerateNetworkPoliciesRequest) (generated []*storage.NetworkPolicy, toDelete []*storage.NetworkPolicyReference, err error) {
	graph, err := g.generateGraph(ctx, req.GetClusterId(), req.GetNetworkDataSince())
	if err != nil {
		return nil, nil, errors.Wrap(err, "generating network graph")
	}
	existingPolicies, toDelete, err := g.getNetworkPolicies(ctx, req.GetDeleteExisting(), req.GetClusterId())
	if err != nil {
		return nil, nil, errors.Wrap(err, "obtaining existing network policies")
	}

	query := search.NewQueryBuilder().AddStrings(search.ClusterID, req.GetClusterId()).ProtoQuery()

	deploymentsQuery, err := search.ParseRawQueryOrEmpty(req.GetQuery())
	if err != nil {
		return nil, nil, errors.Wrap(err, "parsing query")
	}

	var relevantDeploymentIDs set.StringSet
	if deploymentsQuery.Query != nil {
		query = search.ConjunctionQuery(query, deploymentsQuery)

		relevantDeploymentsResult, err := g.deploymentStore.Search(ctx, query)
		if err != nil {
			return nil, nil, errors.Wrap(err, "determining relevant deployments")
		}

		relevantDeploymentIDs = set.NewStringSet(search.ResultsToIDs(relevantDeploymentsResult)...)
	}

	namespaces, err := g.namespacesStore.SearchNamespaces(ctx, query)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not obtain namespaces metadata")
	}

	namespacesByName := createNamespacesByNameMap(namespaces)

	generatedPolicies := g.generatePolicies(graph, relevantDeploymentIDs, namespacesByName, existingPolicies)
	return generatedPolicies, toDelete, nil
}
