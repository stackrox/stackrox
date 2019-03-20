package graph

import (
	"fmt"
	"sort"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/sliceutils"
)

func getAdjacentNodeIDs(node *v1.NetworkNode) []int32 {
	adjacencies := make([]int32, 0, len(node.OutEdges))
	for adj := range node.OutEdges {
		adjacencies = append(adjacencies, adj)
	}
	sort.Slice(adjacencies, func(i, j int) bool { return adjacencies[i] < adjacencies[j] })
	return adjacencies
}

func adjacentNodeIDsToMap(adjacencies []int32) map[int32]*v1.NetworkEdgePropertiesBundle {
	result := make(map[int32]*v1.NetworkEdgePropertiesBundle, len(adjacencies))
	for _, adj := range adjacencies {
		result[adj] = &v1.NetworkEdgePropertiesBundle{}
	}
	return result
}

func getPolicyIDs(node *v1.NetworkNode) []string {
	result := make([]string, len(node.GetPolicyIds()))
	copy(result, node.GetPolicyIds())
	sort.Strings(result)
	return result
}

func computeNodeDiff(oldNode, newNode *v1.NetworkNode) (oldNodeDiff, newNodeDiff *v1.NetworkNodeDiff) {
	oldNodePolicies := getPolicyIDs(oldNode)
	newNodePolicies := getPolicyIDs(newNode)
	oldPolicyIDs, newPolicyIDs := sliceutils.StringDiff(oldNodePolicies, newNodePolicies, func(a, b string) bool { return a < b })
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

	oldAdjacencies := getAdjacentNodeIDs(oldNode)
	newAdjacencies := getAdjacentNodeIDs(newNode)
	removedAdjacencies, addedAdjacencies := sliceutils.Int32Diff(oldAdjacencies, newAdjacencies, func(i, j int32) bool { return i < j })
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
				oldNodeDiff = &v1.NetworkNodeDiff{}
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
	if len(oldGraph.GetNodes()) != len(newGraph.GetNodes()) {
		return nil, nil, fmt.Errorf("graph node counts differ: %d (old) vs %d (new)",
			len(oldGraph.GetNodes()), len(newGraph.GetNodes()))
	}
	added = &v1.NetworkGraphDiff{
		NodeDiffs: make(map[int32]*v1.NetworkNodeDiff),
	}
	removed = &v1.NetworkGraphDiff{
		NodeDiffs: make(map[int32]*v1.NetworkNodeDiff),
	}
	for i, oldNode := range oldGraph.GetNodes() {
		newNode := newGraph.GetNodes()[i]
		oldNodeDiff, newNodeDiff := computeNodeDiff(oldNode, newNode)

		if oldNodeDiff != nil {
			removed.NodeDiffs[int32(i)] = oldNodeDiff
		}
		if newNodeDiff != nil {
			added.NodeDiffs[int32(i)] = newNodeDiff
		}
	}
	return
}
