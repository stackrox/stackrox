package service

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/protocompat"
	"google.golang.org/protobuf/proto"
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
	node = &v1.NetworkNode{}
	node.SetEntity(entity.ToProto())
	node.SetOutEdges(make(map[int32]*v1.NetworkEdgePropertiesBundle))
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
		nd := &storage.NetworkEntityInfo_Deployment{}
		nd.SetName(deployment.GetName())
		nd.SetNamespace(deployment.GetNamespace())
		nd.SetCluster(deployment.GetCluster())
		node.GetEntity().SetDeployment(proto.ValueOrDefault(nd))
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
			// DO NOT SUBMIT: Migrate the direct oneof field access (go/go-opaque-special-cases/oneof.md).
			srcNode.GetEntity().Desc = props.GetSrcEntity().GetDesc()
		}

		if props.GetDstEntity().GetType() == storage.NetworkEntityInfo_LISTEN_ENDPOINT {
			if deployment := srcNode.GetEntity().GetDeployment(); deployment != nil {
				ndl := &storage.NetworkEntityInfo_Deployment_ListenPort{}
				ndl.SetPort(props.GetDstPort())
				ndl.SetL4Protocol(props.GetL4Protocol())
				deployment.SetListenPorts(append(deployment.GetListenPorts(), ndl))
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
			// DO NOT SUBMIT: Migrate the direct oneof field access (go/go-opaque-special-cases/oneof.md).
			dstNode.GetEntity().Desc = props.GetDstEntity().GetDesc()
		}

		tgtIdx := int32(dstIdx)

		tgtEdgeBundle := srcNode.GetOutEdges()[tgtIdx]
		if tgtEdgeBundle == nil {
			tgtEdgeBundle = &v1.NetworkEdgePropertiesBundle{}
			srcNode.GetOutEdges()[tgtIdx] = tgtEdgeBundle
		}

		edgeProps := &v1.NetworkEdgeProperties{}
		edgeProps.SetPort(props.GetDstPort())
		edgeProps.SetProtocol(props.GetL4Protocol())

		edgeProps.SetLastActiveTimestamp(flow.GetLastSeenTimestamp())
		if edgeProps.GetLastActiveTimestamp() == nil {
			edgeProps.SetLastActiveTimestamp(protocompat.TimestampNow())
		}

		tgtEdgeBundle.SetProperties(append(tgtEdgeBundle.GetProperties(), edgeProps))
	}
}

func (b *flowGraphBuilder) Build() *v1.NetworkGraph {
	ng := &v1.NetworkGraph{}
	ng.SetNodes(b.nodes)
	return ng
}
