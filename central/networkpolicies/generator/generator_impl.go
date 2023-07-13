package generator

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	dDS "github.com/stackrox/rox/central/deployment/datastore"
	nsDS "github.com/stackrox/rox/central/namespace/datastore"
	networkBaselineDataStore "github.com/stackrox/rox/central/networkbaseline/datastore"
	"github.com/stackrox/rox/central/networkgraph/aggregator"
	"github.com/stackrox/rox/central/networkgraph/entity/networktree"
	nfDS "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	npDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/networkgraph/tree"
	"github.com/stackrox/rox/pkg/objects"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	generatedNetworkPolicyLabel         = `network-policy-generator.stackrox.io/generated`
	baselineGeneratedNetworkPolicyLabel = `network-policy-generator.stackrox.io/from-baseline`

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
	networkTreeMgr      networktree.Manager
	namespacesStore     nsDS.DataStore
	globalFlowDataStore nfDS.ClusterDataStore
	networkBaselines    networkBaselineDataStore.ReadOnlyDataStore
}

func markGeneratedPoliciesForDeletion(policies []*storage.NetworkPolicy) ([]*storage.NetworkPolicy, []*storage.NetworkPolicyReference) {
	var userPolicies []*storage.NetworkPolicy
	var toDelete []*storage.NetworkPolicyReference

	for _, policy := range policies {
		if isProtectedNamespace(policy.GetNamespace()) {
			continue
		}
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
		if isProtectedNamespace(policy.GetNamespace()) {
			continue
		}
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

func (g *generator) generateGraph(ctx context.Context, clusterID string, query *v1.Query, since *types.Timestamp, includePorts bool) (map[networkgraph.Entity]*node, error) {
	// Temporarily elevate permissions to obtain all network flows in cluster.
	networkGraphGenElevatedCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedClusterLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph),
			sac.ClusterScopeKeys(clusterID)))

	clusterFlowStore, err := g.globalFlowDataStore.GetFlowStore(networkGraphGenElevatedCtx, clusterID)
	if err != nil {
		return nil, err
	} else if clusterFlowStore == nil {
		return nil, errors.Errorf("could not obtain flow store for cluster %q", clusterID)
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
	relevantDeployments := sac.FilterSlice(networkFlowsChecker, deployments, func(deployment *storage.Deployment) sac.ScopePredicate {
		return sac.ScopeSuffix{sac.NamespaceScopeKey(deployment.GetNamespace())}
	})
	relevantDeploymentsMap := objects.ListDeploymentsMapByIDFromDeployments(relevantDeployments)

	// Since we are generating ingress policies only, retrieve all flows incoming to one of the relevant deployments.
	// Note that this will never retrieve listen endpoint "flows".
	// TODO(ROX-???): this needs to be changed should we ever generate egress policies!
	flows, _, err := clusterFlowStore.GetMatchingFlows(networkGraphGenElevatedCtx, func(flowProps *storage.NetworkFlowProperties) bool {
		dstEnt := flowProps.GetDstEntity()
		return dstEnt.GetType() == storage.NetworkEntityInfo_DEPLOYMENT && relevantDeploymentsMap[dstEnt.GetId()] != nil
	}, since)
	if err != nil {
		return nil, errors.Wrapf(err, "could not obtain network flow information for cluster %q", clusterID)
	}

	networkTree := tree.NewMultiNetworkTree(
		g.networkTreeMgr.GetReadOnlyNetworkTree(ctx, clusterID),
		g.networkTreeMgr.GetDefaultNetworkTree(ctx),
	)

	// Aggregate all external conns into supernet conns for which external entities do not exists (as a result of deletion).
	aggr, err := aggregator.NewSubnetToSupernetConnAggregator(networkTree)
	utils.Should(err)
	flows = aggr.Aggregate(flows)
	flows, missingInfoFlows := networkgraph.UpdateFlowsWithEntityDesc(flows, objects.ListDeploymentsMapByIDFromDeployments(relevantDeployments),
		func(id string) *storage.NetworkEntityInfo {
			if networkTree == nil {
				return nil
			}
			return networkTree.Get(id)
		},
	)

	// Aggregate all external flows by node names to control the number of external nodes.
	flows = aggregator.NewDuplicateNameExtSrcConnAggregator().Aggregate(flows)
	missingInfoFlows = aggregator.NewDuplicateNameExtSrcConnAggregator().Aggregate(missingInfoFlows)
	return g.buildGraph(ctx, clusterID, relevantDeployments, flows, missingInfoFlows, includePorts)
}

func (g *generator) populateNode(elevatedCtx context.Context, id string, entityType storage.NetworkEntityInfo_Type) *node {
	n := createNode(networkgraph.Entity{Type: entityType, ID: id})
	if entityType == storage.NetworkEntityInfo_DEPLOYMENT {
		nodeDeployment, ok, err := g.deploymentStore.GetDeployment(elevatedCtx, id)
		if err != nil || !ok {
			// Deployment not found. It might be deleted.
			log.Debugf("detected peer deployment %q missing while trying to generate baseline generated policy", id)
			return nil
		}
		n.deployment = nodeDeployment
	}
	return n
}

func generatePolicy(node *node, namespacesByName map[string]*storage.NamespaceMetadata, ingressPolicies, _ map[string][]*storage.NetworkPolicy) *storage.NetworkPolicy {
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

	policy.Spec.Ingress = generateIngressRules(node, namespacesByName)
	policy.Spec.PolicyTypes = []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE}

	return policy
}

func (g *generator) getBaselineGeneratedPolicyName(deploymentName string) string {
	return fmt.Sprintf("stackrox-baseline-generated-%s", deploymentName)
}

func (g *generator) getBaselineGeneratedPolicy(node *node, namespacesByName map[string]*storage.NamespaceMetadata) *storage.NetworkPolicy {

	policy := &storage.NetworkPolicy{
		Name:        g.getBaselineGeneratedPolicyName(node.deployment.GetName()),
		Namespace:   node.deployment.GetNamespace(),
		ClusterId:   node.deployment.GetClusterId(),
		ClusterName: node.deployment.GetClusterName(),
		Labels: map[string]string{
			baselineGeneratedNetworkPolicyLabel: "true",
		},
		ApiVersion: networkPolicyAPIVersion,
		Spec: &storage.NetworkPolicySpec{
			PodSelector: labelSelectorForDeployment(node.deployment),
		},
	}

	policy.Spec.Ingress = generateIngressRules(node, namespacesByName)
	policy.Spec.PolicyTypes = []storage.NetworkPolicyType{storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE}

	return policy
}

func (g *generator) generatePolicies(graph map[networkgraph.Entity]*node, namespacesByName map[string]*storage.NamespaceMetadata, existingPolicies []*storage.NetworkPolicy) []*storage.NetworkPolicy {
	ingressPolicies, egressPolicies := groupNetworkPolicies(existingPolicies)

	var generatedPolicies []*storage.NetworkPolicy
	for _, node := range graph {
		if !node.selected || node.deployment == nil {
			continue
		}
		if isProtectedDeployment(node.deployment) {
			continue
		}

		policy := generatePolicy(node, namespacesByName, ingressPolicies, egressPolicies)
		if policy != nil {
			generatedPolicies = append(generatedPolicies, policy)
		}
	}

	return generatedPolicies
}

func (g *generator) getNamespacesByName(ctx context.Context, clusterID string) (map[string]*storage.NamespaceMetadata, error) {
	clusterIDQuery := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	namespaces, err := g.namespacesStore.SearchNamespaces(ctx, clusterIDQuery)
	if err != nil {
		return nil, errors.Wrap(err, "could not obtain namespaces metadata")
	}

	return createNamespacesByNameMap(namespaces), nil
}

func (g *generator) Generate(ctx context.Context, req *v1.GenerateNetworkPoliciesRequest) (generated []*storage.NetworkPolicy, toDelete []*storage.NetworkPolicyReference, err error) {
	parsedQuery, err := search.ParseQuery(req.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not parse query")
	}

	graph, err := g.generateGraph(ctx, req.GetClusterId(), parsedQuery, req.GetNetworkDataSince(), req.GetIncludePorts())
	if err != nil {
		return nil, nil, errors.Wrap(err, "generating network graph")
	}

	existingPolicies, toDelete, err := g.getNetworkPolicies(ctx, req.GetDeleteExisting(), req.GetClusterId())
	if err != nil {
		return nil, nil, errors.Wrap(err, "obtaining existing network policies")
	}

	namespacesByName, err := g.getNamespacesByName(ctx, req.GetClusterId())
	if err != nil {
		return nil, nil, err
	}

	generatedPolicies := g.generatePolicies(graph, namespacesByName, existingPolicies)
	return generatedPolicies, toDelete, nil
}

func (g *generator) GenerateFromBaselineForDeployment(
	ctx context.Context,
	req *v1.GetBaselineGeneratedPolicyForDeploymentRequest,
) ([]*storage.NetworkPolicy, []*storage.NetworkPolicyReference, error) {
	deployment, ok, err := g.deploymentStore.GetDeployment(ctx, req.GetDeploymentId())
	if err != nil {
		return nil, nil, err
	} else if !ok {
		return nil, nil, errors.New("deployment not found")
	}

	node, err := g.generateNodeFromBaselineForDeployment(ctx, deployment, req.GetIncludePorts())
	if err != nil {
		return nil, nil, err
	} else if node == nil {
		return nil, nil, errors.New("failed to generate graph node for this deployment")
	}

	namespacesByName, err := g.getNamespacesByName(ctx, deployment.GetClusterId())
	if err != nil {
		return nil, nil, err
	}

	// Generate network policy for this deployment
	policy := g.getBaselineGeneratedPolicy(node, namespacesByName)

	// We mark the policy as `toDelete` to signal to sensor that it should replace the policy if it exists already.
	// If it doesn't exist, sensor will create the policy.
	// This is because we keep exactly one baseline-generated network policy per deployment, and we make it comprehensive.
	// This allows us to ensure, e.g, that deletions from a baseline reflect in the network policy.
	// It also makes the end state idempotent and not path-dependent.
	return []*storage.NetworkPolicy{policy}, []*storage.NetworkPolicyReference{{Name: policy.GetName(), Namespace: policy.GetNamespace()}}, nil
}
