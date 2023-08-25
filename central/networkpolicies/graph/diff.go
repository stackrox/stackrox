package graph

import (
	"sort"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/utils"
)

func getAdjacentNodeIDs(adjacencies map[string]*v1.NetworkEdgePropertiesBundle) []string {
	ids := make([]string, 0, len(adjacencies))
	for id := range adjacencies {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func adjacentNodeIDsToMap(adjacencies []string) map[string]*v1.NetworkEdgePropertiesBundle {
	result := make(map[string]*v1.NetworkEdgePropertiesBundle, len(adjacencies))
	for _, adj := range adjacencies {
		result[adj] = &v1.NetworkEdgePropertiesBundle{}
	}
	return result
}

func getPolicyIDs(node *v1.NetworkNode) []string {
	result := sliceutils.ShallowClone(node.GetPolicyIds())
	sort.Strings(result)
	return result
}

func computeNodeDiff(oldG, newG *networkGraphWrapper, id string) (oldNodeDiff, newNodeDiff *v1.NetworkNodeDiff) {
	oldNode := oldG.getNode(id)
	newNode := newG.getNode(id)
	oldNodePolicies := getPolicyIDs(oldNode)
	newNodePolicies := getPolicyIDs(newNode)
	oldPolicyIDs, newPolicyIDs := sliceutils.Diff(oldNodePolicies, newNodePolicies, func(a, b string) bool { return a < b })
	if len(oldPolicyIDs) > 0 {
		oldNodeDiff = &v1.NetworkNodeDiff{
			PolicyIds: oldPolicyIDs,
		}
	}
	if len(newPolicyIDs) > 0 {
		newNodeDiff = &v1.NetworkNodeDiff{
			PolicyIds: newPolicyIDs,
		}
	}

	oldAdjacencies := getAdjacentNodeIDs(oldG.getNodeOutEdges(id))
	newAdjacencies := getAdjacentNodeIDs(newG.getNodeOutEdges(id))
	removedAdjacencies, addedAdjacencies := sliceutils.Diff(oldAdjacencies, newAdjacencies, func(a, b string) bool { return a < b })
	if len(removedAdjacencies) > 0 {
		if oldNodeDiff == nil {
			oldNodeDiff = &v1.NetworkNodeDiff{}
		}
		oldNodeDiff.OutEdges = adjacentNodeIDsToMap(removedAdjacencies)
	}
	if len(addedAdjacencies) > 0 {
		if newNodeDiff == nil {
			newNodeDiff = &v1.NetworkNodeDiff{}
		}
		newNodeDiff.OutEdges = adjacentNodeIDsToMap(addedAdjacencies)
	}

	if oldNode.GetNonIsolatedIngress() != newNode.GetNonIsolatedIngress() {
		if oldNode.GetNonIsolatedIngress() {
			if oldNodeDiff == nil {
				oldNodeDiff = &v1.NetworkNodeDiff{}
			}
			oldNodeDiff.NonIsolatedIngress = true
		} else {
			if newNodeDiff == nil {
				newNodeDiff = &v1.NetworkNodeDiff{}
			}
			newNodeDiff.NonIsolatedIngress = true
		}
	}
	if oldNode.GetNonIsolatedEgress() != newNode.GetNonIsolatedEgress() {
		if oldNode.GetNonIsolatedEgress() {
			if oldNodeDiff == nil {
				oldNodeDiff = &v1.NetworkNodeDiff{}
			}
			oldNodeDiff.NonIsolatedEgress = true
		} else {
			if newNodeDiff == nil {
				newNodeDiff = &v1.NetworkNodeDiff{}
			}
			newNodeDiff.NonIsolatedEgress = true
		}
	}
	return
}

// ComputeDiff computes a diff between the old and the new graph.
func ComputeDiff(oldGraph, newGraph *v1.NetworkGraph) (removed, added *v1.NetworkGraphDiff, _ error) {
	added = &v1.NetworkGraphDiff{
		NodeDiffs: make(map[string]*v1.NetworkNodeDiff),
	}
	removed = &v1.NetworkGraphDiff{
		NodeDiffs: make(map[string]*v1.NetworkNodeDiff),
	}

	oldG := newNetworkGraph(oldGraph)
	newG := newNetworkGraph(newGraph)

	for id := range newG.graphNodes {
		if oldNode := oldG.getNode(id); oldNode == nil {
			added.NodeDiffs[id] = newG.getNetworkNodeDiffProto(id)
		}
	}

	for id := range oldG.graphNodes {
		if newNode := newG.getNode(id); newNode == nil {
			removed.NodeDiffs[id] = oldG.getNetworkNodeDiffProto(id)
			continue
		}
		oldNodeDiff, newNodeDiff := computeNodeDiff(oldG, newG, id)
		if oldNodeDiff != nil {
			removed.NodeDiffs[id] = oldNodeDiff
		}
		if newNodeDiff != nil {
			added.NodeDiffs[id] = newNodeDiff
		}
	}
	return
}

type networkGraphWrapper struct {
	graphNodes map[string]*v1.NetworkNode
	idxToIDs   []string
}

func newNetworkGraph(g *v1.NetworkGraph) *networkGraphWrapper {
	graphNodes := make(map[string]*v1.NetworkNode)
	idxToIDs := make([]string, len(g.GetNodes()))
	for i, node := range g.GetNodes() {
		idxToIDs[i] = node.GetEntity().GetId()
		graphNodes[node.GetEntity().GetId()] = node
	}

	return &networkGraphWrapper{
		graphNodes: graphNodes,
		idxToIDs:   idxToIDs,
	}
}

func (g *networkGraphWrapper) getNode(id string) *v1.NetworkNode {
	return g.graphNodes[id]
}

func (g *networkGraphWrapper) getNodeOutEdges(id string) map[string]*v1.NetworkEdgePropertiesBundle {
	node := g.graphNodes[id]
	if node == nil {
		return nil
	}

	ret := make(map[string]*v1.NetworkEdgePropertiesBundle)
	for i, edge := range node.GetOutEdges() {
		if i < 0 || i >= int32(len(g.idxToIDs)) {
			utils.Should(errors.Errorf("invalid network graph node %d index", i))
			continue
		}
		ret[g.idxToIDs[i]] = edge
	}
	return ret
}

func (g *networkGraphWrapper) getNetworkNodeDiffProto(id string) *v1.NetworkNodeDiff {
	node := g.graphNodes[id]
	if node == nil {
		return nil
	}

	return &v1.NetworkNodeDiff{
		PolicyIds:          getPolicyIDs(node),
		OutEdges:           g.getNodeOutEdges(id),
		NonIsolatedIngress: node.GetNonIsolatedIngress(),
		NonIsolatedEgress:  node.GetNonIsolatedEgress(),
	}
}
