package generator

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkentity"
)

type node struct {
	entity     networkentity.Entity
	deployment *storage.Deployment
	incoming   map[*node]struct{}
	outgoing   map[*node]struct{}
}

func createNode(entity networkentity.Entity) *node {
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

func buildGraph(deployments []*storage.Deployment, allFlows []*storage.NetworkFlow) map[networkentity.Entity]*node {
	nodesByKey := make(map[networkentity.Entity]*node)

	for _, flow := range allFlows {
		srcKey := networkentity.FromProto(flow.GetProps().GetSrcEntity())
		srcNode := nodesByKey[srcKey]
		if srcNode == nil {
			srcNode = createNode(srcKey)
			nodesByKey[srcKey] = srcNode
		}

		dstKey := networkentity.FromProto(flow.GetProps().GetDstEntity())
		dstNode := nodesByKey[dstKey]
		if dstNode == nil {
			dstNode = createNode(dstKey)
			nodesByKey[dstKey] = dstNode
		}

		srcNode.outgoing[dstNode] = struct{}{}
		dstNode.incoming[srcNode] = struct{}{}
	}

	for _, deployment := range deployments {
		key := networkentity.Entity{
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

	return nodesByKey
}
