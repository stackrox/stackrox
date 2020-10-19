package service

import (
	"github.com/gogo/protobuf/types"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
)

type flowGraphBuilder struct {
	nodes       []*v1.NetworkNode
	nodeIndices map[networkgraph.Entity]int
}

func newFlowGraphBuilder() *flowGraphBuilder {
	return &flowGraphBuilder{
		nodeIndices: make(map[networkgraph.Entity]int),
	}
}

func (b *flowGraphBuilder) removeLastNode() {
	lastNodeIdx := len(b.nodes) - 1
	if lastNodeIdx < 0 {
		return
	}
	b.nodes = b.nodes[:lastNodeIdx]
	for key, idx := range b.nodeIndices {
		if idx == lastNodeIdx {
			delete(b.nodeIndices, key)
			break
		}
	}
}

func (b *flowGraphBuilder) getNode(entity networkgraph.Entity, addIfMissing bool) (idx int, node *v1.NetworkNode, added bool) {
	idx, found := b.nodeIndices[entity]
	if found {
		return idx, b.nodes[idx], false
	}
	if !addIfMissing {
		return -1, nil, false
	}
	node = &v1.NetworkNode{
		Entity:   entity.ToProto(),
		OutEdges: make(map[int32]*v1.NetworkEdgePropertiesBundle),
	}
	idx = len(b.nodes)
	b.nodes = append(b.nodes, node)
	b.nodeIndices[entity] = idx
	return idx, node, true
}

func (b *flowGraphBuilder) AddDeployments(deployments []*storage.ListDeployment) {
	for _, deployment := range deployments {
		key := networkgraph.Entity{
			Type: storage.NetworkEntityInfo_DEPLOYMENT,
			ID:   deployment.GetId(),
		}
		_, node, added := b.getNode(key, true)
		if !added {
			continue
		}
		node.Entity.Desc = &storage.NetworkEntityInfo_Deployment_{
			Deployment: &storage.NetworkEntityInfo_Deployment{
				Name:      deployment.GetName(),
				Namespace: deployment.GetNamespace(),
				Cluster:   deployment.GetCluster(),
			},
		}
	}
}

func (b *flowGraphBuilder) AddFlows(flows []*storage.NetworkFlow) {
	for _, flow := range flows {
		props := flow.GetProps()
		srcEnt := networkgraph.EntityFromProto(props.GetSrcEntity())
		// Deployment accessible by request scope are already added as nodes before adding the flows, hence we skip adding them again.
		_, srcNode, added := b.getNode(srcEnt, srcEnt.Type != storage.NetworkEntityInfo_DEPLOYMENT)
		if srcNode == nil {
			continue
		}

		if networkgraph.IsExternal(props.GetSrcEntity()) {
			srcNode.Entity.Desc = props.GetSrcEntity().GetDesc()
		}

		if props.GetDstEntity().GetType() == storage.NetworkEntityInfo_LISTEN_ENDPOINT {
			if deployment := srcNode.Entity.GetDeployment(); deployment != nil {
				deployment.ListenPorts = append(deployment.ListenPorts, &storage.NetworkEntityInfo_Deployment_ListenPort{
					Port:       props.GetDstPort(),
					L4Protocol: props.GetL4Protocol(),
				})
			} else if added {
				log.Errorf("UNEXPECTED: Listen endpoint for non-deployment source entity %v", srcEnt)
				b.removeLastNode()
			}
			continue
		}

		dstEnt := networkgraph.EntityFromProto(props.GetDstEntity())
		dstIdx, dstNode, _ := b.getNode(dstEnt, dstEnt.Type != storage.NetworkEntityInfo_DEPLOYMENT)
		// For non-deployment nodes, if the destination deployment is not accessible, remove this source node.
		if dstIdx == -1 {
			if added {
				b.removeLastNode()
			}
			continue
		}

		if networkgraph.IsExternal(props.GetDstEntity()) {
			dstNode.Entity.Desc = props.GetDstEntity().GetDesc()
		}

		tgtIdx := int32(dstIdx)

		tgtEdgeBundle := srcNode.OutEdges[tgtIdx]
		if tgtEdgeBundle == nil {
			tgtEdgeBundle = &v1.NetworkEdgePropertiesBundle{}
			srcNode.OutEdges[tgtIdx] = tgtEdgeBundle
		}

		edgeProps := &v1.NetworkEdgeProperties{
			Port:     props.GetDstPort(),
			Protocol: props.L4Protocol,
		}

		edgeProps.LastActiveTimestamp = flow.GetLastSeenTimestamp()
		if edgeProps.LastActiveTimestamp == nil {
			edgeProps.LastActiveTimestamp = types.TimestampNow()
		}

		tgtEdgeBundle.Properties = append(tgtEdgeBundle.Properties, edgeProps)
	}
}

func (b *flowGraphBuilder) Build() *v1.NetworkGraph {
	return &v1.NetworkGraph{
		Nodes: b.nodes,
	}
}
