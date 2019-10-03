package generator

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	dDS "github.com/stackrox/rox/central/deployment/datastore"
	nsDS "github.com/stackrox/rox/central/namespace/datastore"
	"github.com/stackrox/rox/central/networkflow"
	nfDS "github.com/stackrox/rox/central/networkflow/datastore"
	npDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/objects"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
)

const (
	generatedNetworkPolicyLabel = `network-policy-generator.stackrox.io/generated`

	networkPolicyAPIVersion = `networking.k8s.io/v1`
)

var (
	log = logging.LoggerForModule()

	networkFlowsSAC = sac.ForResource(resources.NetworkGraph)
)

func isGeneratedPolicy(policy *storage.NetworkPolicy) bool {
	_, ok := policy.GetLabels()[generatedNetworkPolicyLabel]
	return ok
}

type generator struct {
	networkPolicies     npDS.DataStore
	deploymentStore     dDS.DataStore
	namespacesStore     nsDS.DataStore
	globalFlowDataStore nfDS.ClusterDataStore
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

func (g *generator) generateGraph(ctx context.Context, clusterID string, query *v1.Query, since *types.Timestamp) (map[networkgraph.Entity]*node, error) {
	// Temporarily elevate permissions to obtain all network flows in cluster.
	networkGraphGenElevatedCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph),
			sac.ClusterScopeKeys(clusterID)))

	clusterFlowStore := g.globalFlowDataStore.GetFlowStore(networkGraphGenElevatedCtx, clusterID)
	if clusterFlowStore == nil {
		return nil, fmt.Errorf("could not obtain flow store for cluster %q", clusterID)
	}

	clusterIDQuery := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	deploymentsQuery := clusterIDQuery
	if query.GetQuery() != nil {
		deploymentsQuery = search.ConjunctionQuery(deploymentsQuery, query)
	}
	deployments, err := g.deploymentStore.SearchRawDeployments(ctx, deploymentsQuery)
	if err != nil {
		return nil, errors.Wrapf(err, "could not obtain deployments for cluster %q", clusterID)
	}

	// Filter out only those deployments for which we can see network flows. We cannot reliably generate network
	// policies for other deployments.
	networkFlowsChecker := networkFlowsSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).ClusterID(clusterID)
	filteredSlice, err := sac.FilterSliceReflect(ctx, networkFlowsChecker, deployments, func(deployment *storage.Deployment) sac.ScopePredicate {
		return sac.ScopeSuffix{sac.NamespaceScopeKey(deployment.GetNamespace())}
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not determine network flow access for deployments")
	}
	relevantDeployments := filteredSlice.([]*storage.Deployment)
	relevantDeploymentsMap := objects.ListDeploymentsMapByIDFromDeployments(relevantDeployments)

	// Since we are generating ingress policies only, retrieve all flows incoming to one of the relevant deployments.
	// TODO(ROX-???): this needs to be changed should we ever generate egress policies!
	flows, _, err := clusterFlowStore.GetMatchingFlows(networkGraphGenElevatedCtx, func(flowProps *storage.NetworkFlowProperties) bool {
		dstEnt := flowProps.GetDstEntity()
		return dstEnt.GetType() == storage.NetworkEntityInfo_DEPLOYMENT && relevantDeploymentsMap[dstEnt.GetId()] != nil
	}, since)
	if err != nil {
		return nil, errors.Wrapf(err, "could not obtain network flow information for cluster %q", clusterID)
	}

	okFlows, missingInfoFlows := networkflow.UpdateFlowsWithDeployments(flows, objects.ListDeploymentsMapByIDFromDeployments(relevantDeployments))

	return g.buildGraph(ctx, clusterID, relevantDeployments, okFlows, missingInfoFlows)
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

func (g *generator) generatePolicies(graph map[networkgraph.Entity]*node, namespacesByName map[string]*storage.NamespaceMetadata, existingPolicies []*storage.NetworkPolicy) []*storage.NetworkPolicy {
	ingressPolicies, egressPolicies := groupNetworkPolicies(existingPolicies)

	var generatedPolicies []*storage.NetworkPolicy
	for _, node := range graph {
		if !node.selected || node.deployment == nil {
			continue
		}
		if isSystemDeployment(node.deployment) {
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
	parsedQuery, err := search.ParseQuery(req.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not parse query")
	}

	graph, err := g.generateGraph(ctx, req.GetClusterId(), parsedQuery, req.GetNetworkDataSince())
	if err != nil {
		return nil, nil, errors.Wrap(err, "generating network graph")
	}

	existingPolicies, toDelete, err := g.getNetworkPolicies(ctx, req.GetDeleteExisting(), req.GetClusterId())
	if err != nil {
		return nil, nil, errors.Wrap(err, "obtaining existing network policies")
	}

	clusterIDQuery := search.NewQueryBuilder().AddExactMatches(search.ClusterID, req.GetClusterId()).ProtoQuery()
	namespaces, err := g.namespacesStore.SearchNamespaces(ctx, clusterIDQuery)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not obtain namespaces metadata")
	}

	namespacesByName := createNamespacesByNameMap(namespaces)

	generatedPolicies := g.generatePolicies(graph, namespacesByName, existingPolicies)
	return generatedPolicies, toDelete, nil
}
