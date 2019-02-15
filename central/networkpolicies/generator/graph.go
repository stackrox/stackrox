package generator

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkentity"
)

type node struct {
	entity     networkentity.Entity
	deployment *storage.Deployment
	incoming   []*node
	outgoing   []*node
}

func createNode(entity networkentity.Entity) *node {
	return &node{
		entity: entity,
	}
}

func (n *node) hasInternetIngress() bool {
	for _, srcNode := range n.incoming {
		if srcNode.entity.Type == storage.NetworkEntityInfo_INTERNET {
			return true
		}
	}

	for _, container := range n.deployment.GetContainers() {
		for _, port := range container.GetPorts() {
			if port.GetExposure() == storage.PortConfig_NODE || port.GetExposure() == storage.PortConfig_EXTERNAL {
				return true
			}
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

		srcNode.outgoing = append(srcNode.outgoing, dstNode)
		dstNode.incoming = append(dstNode.incoming, srcNode)
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
