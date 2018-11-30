package service

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/networkentity"
)

type flowGraphBuilder struct {
	nodes       []*v1.NetworkNode
	nodeIndices map[networkentity.Entity]int
}

func newFlowGraphBuilder() *flowGraphBuilder {
	return &flowGraphBuilder{
		nodeIndices: make(map[networkentity.Entity]int),
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

func (b *flowGraphBuilder) getNode(entity networkentity.Entity, addIfMissing bool) (idx int, node *v1.NetworkNode, added bool) {
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

func (b *flowGraphBuilder) AddDeployments(deployments []*v1.Deployment) {
	for _, deployment := range deployments {
		key := networkentity.Entity{
			Type: v1.NetworkEntityInfo_DEPLOYMENT,
			ID:   deployment.GetId(),
		}
		_, node, added := b.getNode(key, true)
		if !added {
			continue
		}
		node.Entity.Desc = &v1.NetworkEntityInfo_Deployment_{
			Deployment: &v1.NetworkEntityInfo_Deployment{
				Name:      deployment.GetName(),
				Namespace: deployment.GetNamespace(),
				Cluster:   deployment.GetClusterName(),
			},
		}
	}
}

func (b *flowGraphBuilder) AddFlows(flows []*v1.NetworkFlow) {
	for _, flow := range flows {
		props := flow.GetProps()
		srcEnt := networkentity.FromProto(props.GetSrcEntity())
		_, srcNode, added := b.getNode(srcEnt, srcEnt.Type != v1.NetworkEntityInfo_DEPLOYMENT)
		if srcNode == nil {
			continue
		}
		dstEnt := networkentity.FromProto(props.GetDstEntity())
		dstIdx, _, _ := b.getNode(dstEnt, dstEnt.Type != v1.NetworkEntityInfo_DEPLOYMENT)
		if dstIdx == -1 {
			if added {
				b.removeLastNode()
			}
			continue
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
