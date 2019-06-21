package generator

import (
	"context"

	networkFlow "github.com/stackrox/rox/central/networkflow/service"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/set"
)

type node struct {
	entity     networkgraph.Entity
	masked     bool
	deployment *storage.Deployment
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

func buildGraph(ctx context.Context, clusterID string, deployments []*storage.Deployment, allFlows []*storage.NetworkFlow) (map[networkgraph.Entity]*node, error) {
	nodesByKey := make(map[networkgraph.Entity]*node)
	filteredFlows, maskedDeployments, err := networkFlow.FilterFlowsAndMaskScopeAlienDeployments(ctx, clusterID, allFlows, deployments)
	if err != nil {
		return nil, err
	}
	maskedDeploymentSet := set.NewStringSet()
	for _, d := range maskedDeployments {
		maskedDeploymentSet.Add(d.GetId())
	}

	for _, flow := range filteredFlows {
		srcKey := networkgraph.EntityFromProto(flow.GetProps().GetSrcEntity())
		srcNode := nodesByKey[srcKey]
		if srcNode == nil {
			srcNode = createNode(srcKey)
			if maskedDeploymentSet.Contains(srcNode.entity.ID) {
				srcNode.masked = true
			}
			nodesByKey[srcKey] = srcNode
		}

		dstKey := networkgraph.EntityFromProto(flow.GetProps().GetDstEntity())
		dstNode := nodesByKey[dstKey]
		if dstNode == nil {
			dstNode = createNode(dstKey)
			if maskedDeploymentSet.Contains(dstNode.entity.ID) {
				dstNode.masked = true
			}
			nodesByKey[dstKey] = dstNode
		}

		srcNode.outgoing[dstNode] = struct{}{}
		dstNode.incoming[srcNode] = struct{}{}
	}

	// These deployments are visible (and exist in query)
	for _, deployment := range deployments {
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
	}

	return nodesByKey, nil
}
