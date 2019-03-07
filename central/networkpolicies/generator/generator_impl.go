package generator

import (
	"fmt"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/deployment/datastore"
	flowStore "github.com/stackrox/rox/central/networkflow/store"
	"github.com/stackrox/rox/central/networkpolicies/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkentity"
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

type annotatedNode struct {
	node       *v1.NetworkNode
	deployment *storage.Deployment
	incoming   []*annotatedNode
	flowStore.FlowStore
}

type generator struct {
	networkPolicyStore store.Store
	deploymentStore    datastore.DataStore
	globalFlowStore    flowStore.ClusterStore
}

func markGeneratedPoliciesForDeletion(policies []*storage.NetworkPolicy) ([]*storage.NetworkPolicy, []*v1.NetworkPolicyReference) {
	var userPolicies []*storage.NetworkPolicy
	var toDelete []*v1.NetworkPolicyReference

	for _, policy := range policies {
		if isGeneratedPolicy(policy) {
			toDelete = append(toDelete, &v1.NetworkPolicyReference{
				Name:      policy.GetName(),
				Namespace: policy.GetNamespace(),
			})
		} else {
			userPolicies = append(userPolicies, policy)
		}
	}

	return userPolicies, toDelete
}

func markAllPoliciesForDeletion(policies []*storage.NetworkPolicy) []*v1.NetworkPolicyReference {
	toDelete := make([]*v1.NetworkPolicyReference, 0, len(policies))
	for _, policy := range policies {
		toDelete = append(toDelete, &v1.NetworkPolicyReference{
			Name:      policy.GetName(),
			Namespace: policy.GetNamespace(),
		})
	}
	return toDelete
}

func (g *generator) getNetworkPolicies(deleteExistingMode v1.GenerateNetworkPoliciesRequest_DeleteExistingPoliciesMode, clusterID string) ([]*storage.NetworkPolicy, []*v1.NetworkPolicyReference, error) {
	policies, err := g.networkPolicyStore.GetNetworkPolicies(clusterID, "")
	if err != nil {
		return nil, nil, fmt.Errorf("obtaining network policies: %v", err)
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

func (g *generator) generateGraph(clusterID string, since *types.Timestamp) (map[networkentity.Entity]*node, error) {
	clusterFlowStore := g.globalFlowStore.GetFlowStore(clusterID)
	if clusterFlowStore == nil {
		return nil, fmt.Errorf("could not obtain flow store for cluster %q", clusterID)
	}

	allFlows, _, err := clusterFlowStore.GetAllFlows(since)
	if err != nil {
		return nil, fmt.Errorf("could not obtain network flow information for cluster %q: %v", clusterID, err)
	}

	deployments, err := g.deploymentStore.SearchRawDeployments(&v1.Query{
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
		return nil, fmt.Errorf("could not obtain deployments for cluster %q: %v", clusterID, err)
	}

	return buildGraph(deployments, allFlows), nil
}

func generatePolicy(node *node, ingressPolicies, egressPolicies map[string][]*storage.NetworkPolicy) *storage.NetworkPolicy {
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
			PodSelector: node.deployment.GetLabelSelector(),
		},
	}

	ingressRule := generateIngressRule(node)
	if ingressRule != nil {
		policy.Spec.Ingress = append(policy.Spec.Ingress, ingressRule)
		policy.Spec.PolicyTypes = append(policy.Spec.PolicyTypes, storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE)
	}

	return policy
}

func (g *generator) generatePolicies(graph map[networkentity.Entity]*node, deploymentIDs set.StringSet, existingPolicies []*storage.NetworkPolicy) []*storage.NetworkPolicy {
	ingressPolicies, egressPolicies := groupNetworkPolicies(existingPolicies)

	var generatedPolicies []*storage.NetworkPolicy
	for _, node := range graph {
		if node.deployment == nil {
			continue
		}
		if isSystemDeployment(node.deployment) || (deploymentIDs.IsInitialized() && !deploymentIDs.Contains(node.deployment.GetId())) {
			continue
		}

		policy := generatePolicy(node, ingressPolicies, egressPolicies)
		if policy != nil {
			generatedPolicies = append(generatedPolicies, policy)
		}
	}

	return generatedPolicies
}

func (g *generator) Generate(req *v1.GenerateNetworkPoliciesRequest) (generated []*storage.NetworkPolicy, toDelete []*v1.NetworkPolicyReference, err error) {
	graph, err := g.generateGraph(req.GetClusterId(), req.GetNetworkDataSince())
	if err != nil {
		return nil, nil, fmt.Errorf("generating network graph: %v", err)
	}
	existingPolicies, toDelete, err := g.getNetworkPolicies(req.GetDeleteExisting(), req.GetClusterId())
	if err != nil {
		return nil, nil, fmt.Errorf("obtaining existing network policies: %v", err)
	}

	query := search.NewQueryBuilder().AddStrings(search.ClusterID, req.GetClusterId()).ProtoQuery()

	deploymentsQuery, err := search.ParseRawQueryOrEmpty(req.GetQuery())
	if err != nil {
		return nil, nil, fmt.Errorf("parsing query: %v", err)
	}

	var relevantDeploymentIDs set.StringSet
	if deploymentsQuery.Query != nil {
		query = search.ConjunctionQuery(query, deploymentsQuery)

		relevantDeploymentsResult, err := g.deploymentStore.Search(query)
		if err != nil {
			return nil, nil, fmt.Errorf("determining relevant deployments: %v", err)
		}

		relevantDeploymentIDs = set.NewStringSet(search.ResultsToIDs(relevantDeploymentsResult)...)
	}

	generatedPolicies := g.generatePolicies(graph, relevantDeploymentIDs, existingPolicies)
	return generatedPolicies, toDelete, nil
}
