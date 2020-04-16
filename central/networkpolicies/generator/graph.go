package generator

import (
	"context"

	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

type node struct {
	entity     networkgraph.Entity
	deployment *storage.Deployment
	selected   bool
	masked     bool
	incoming   map[*node]struct{}
	outgoing   map[*node]struct{}
}

func createNode(entity networkgraph.Entity) *node {
	return &node{
		entity:   entity,
		incoming: make(map[*node]struct{}),
		outgoing: make(map[*node]struct{}),
	}
}

func (n *node) hasInternetIngress() bool {
	for srcNode := range n.incoming {
		if srcNode.entity.Type == storage.NetworkEntityInfo_INTERNET {
			return true
		}
	}

	for _, port := range n.deployment.GetPorts() {
		if port.GetExposure() == storage.PortConfig_NODE || port.GetExposure() == storage.PortConfig_EXTERNAL {
			return true
		}
	}
	return false
}

func (n *node) hasMaskedPeer() bool {
	for srcNode := range n.incoming {
		if srcNode.masked {
			return true
		}
	}
	return false
}

func (g *generator) buildGraph(ctx context.Context, clusterID string, selectedDeployments []*storage.Deployment, okFlows, missingInfoFlows []*storage.NetworkFlow) (map[networkgraph.Entity]*node, error) {
	// Determine the deployments that are still missing.
	missingDeploymentIDs := set.NewStringSet()
	for _, flow := range missingInfoFlows {
		props := flow.GetProps()
		// It is sufficient to check the source entity, because we know that we have information regarding the
		// destination deployment entity -- otherwise, we would not retrieve the flow in the first place.
		if props.GetSrcEntity().GetType() == storage.NetworkEntityInfo_DEPLOYMENT && props.GetSrcEntity().GetDeployment() == nil {
			missingDeploymentIDs.Add(props.GetSrcEntity().GetId())
		}
	}

	// Not all of these deployments may be visible, but that's okay.
	var unselectedButVisibleDeployments []*storage.Deployment
	if missingDeploymentIDs.Cardinality() > 0 {
		var err error
		unselectedButVisibleDeployments, err = g.deploymentStore.GetDeployments(ctx, missingDeploymentIDs.AsSlice())
		if err != nil {
			return nil, err
		}

		// Retain only deployments that are not visible or deleted.
		for _, deployment := range unselectedButVisibleDeployments {
			missingDeploymentIDs.Remove(deployment.GetId())
		}
	}

	nodesByKey := make(map[networkgraph.Entity]*node)

	allFlows := make([]*storage.NetworkFlow, 0, len(okFlows)+len(missingInfoFlows))
	allFlows = append(allFlows, okFlows...)
	allFlows = append(allFlows, missingInfoFlows...)

	// Add nodes and edges for all flows.
	for _, flow := range allFlows {
		srcKey := networkgraph.EntityFromProto(flow.GetProps().GetSrcEntity())
		srcNode := nodesByKey[srcKey]
		if srcNode == nil {
			srcNode = createNode(srcKey)
			nodesByKey[srcKey] = srcNode
		}

		dstKey := networkgraph.EntityFromProto(flow.GetProps().GetDstEntity())
		dstNode := nodesByKey[dstKey]
		if dstNode == nil {
			dstNode = createNode(dstKey)
			nodesByKey[dstKey] = dstNode
		}

		srcNode.outgoing[dstNode] = struct{}{}
		dstNode.incoming[srcNode] = struct{}{}
	}

	// Populate deployment data for all deployments that we can see (either selected by the query, or relevant to
	// one of selected deployments and visible).
	// Remaining deployments will leave the `deployment` field of the nodes as nil, which will be interpreted as a
	// masked peer.
	for _, deployment := range selectedDeployments {
		key := networkgraph.Entity{
			Type: storage.NetworkEntityInfo_DEPLOYMENT,
			ID:   deployment.GetId(),
		}

		deploymentNode := nodesByKey[key]
		if deploymentNode == nil {
			deploymentNode = createNode(key)
			nodesByKey[key] = deploymentNode
		}
		deploymentNode.deployment = deployment
		deploymentNode.selected = true
	}

	for _, deployment := range unselectedButVisibleDeployments {
		key := networkgraph.Entity{
			Type: storage.NetworkEntityInfo_DEPLOYMENT,
			ID:   deployment.GetId(),
		}

		deploymentNode := nodesByKey[key]
		if deploymentNode == nil {
			continue
		}
		deploymentNode.deployment = deployment
	}

	if missingDeploymentIDs.Cardinality() > 0 {
		// Finally, do a deployments query with elevated privileges to know which deployments are invisible. These will
		// be then marked as masked.
		// This step exists to ensure that a recently deleted deployment is not interpreted as a masked deployment, which
		// would be extremely bad user experience as the generated policies for its peers would be useless.
		q := search.NewQueryBuilder().AddDocIDSet(missingDeploymentIDs).ProtoQuery()
		viewAllDeploymentsInClusterCtx := sac.WithGlobalAccessScopeChecker(ctx, sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment),
			sac.ClusterScopeKeys(clusterID)))
		results, err := g.deploymentStore.Search(viewAllDeploymentsInClusterCtx, q)
		if err != nil {
			return nil, err
		}
		for _, maskedDeploymentID := range search.ResultsToIDs(results) {
			key := networkgraph.Entity{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				ID:   maskedDeploymentID,
			}

			deploymentNode := nodesByKey[key]
			if deploymentNode == nil {
				continue // shouldn't happen
			}
			deploymentNode.masked = true
		}
	}

	return nodesByKey, nil
}
