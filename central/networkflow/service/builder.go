package service

import (
	"strings"

	"github.com/gogo/protobuf/types"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
)

var (
	l4ProtoByName = map[string]storage.L4Protocol{
		"tcp": storage.L4Protocol_L4_PROTOCOL_TCP,
		"udp": storage.L4Protocol_L4_PROTOCOL_UDP,
	}
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
		_, srcNode, added := b.getNode(srcEnt, srcEnt.Type != storage.NetworkEntityInfo_DEPLOYMENT)
		if srcNode == nil {
			continue
		}
		dstEnt := networkgraph.EntityFromProto(props.GetDstEntity())
		dstIdx, _, _ := b.getNode(dstEnt, dstEnt.Type != storage.NetworkEntityInfo_DEPLOYMENT)
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

func (b *flowGraphBuilder) AddListenPortsFromDeployments(deployments []*storage.Deployment) {
	// TODO(ROX-5301): Remove this and replace with data obtained from collector.
	for _, deployment := range deployments {
		if len(deployment.GetPorts()) == 0 {
			continue
		}

		_, node, _ := b.getNode(networkgraph.EntityForDeployment(deployment.GetId()), false)
		if node == nil {
			continue
		}
		deploymentEntity := node.GetEntity().GetDeployment()
		if deploymentEntity == nil {
			continue
		}

		for _, portCfg := range deployment.GetPorts() {
			listenPort := &storage.NetworkEntityInfo_Deployment_ListenPort{
				Port:       uint32(portCfg.GetContainerPort()),
				L4Protocol: l4ProtoByName[strings.ToLower(portCfg.GetProtocol())],
			}
			deploymentEntity.ListenPorts = append(deploymentEntity.ListenPorts, listenPort)
		}
	}
}

func (b *flowGraphBuilder) Build() *v1.NetworkGraph {
	return &v1.NetworkGraph{
		Nodes: b.nodes,
	}
}
