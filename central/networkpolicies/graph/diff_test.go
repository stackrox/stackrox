package graph

import (
	"sort"
	"testing"

	"github.com/gogo/protobuf/proto"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stretchr/testify/assert"
)

type adjs []string
type pols []string

type nodeSpec struct {
	adjacencies adjs
	policies    pols
}

type nodeSpecMap map[string]nodeSpec

func sortedIDs(ids []string) []string {
	result := sliceutils.ShallowClone(ids)
	sort.Strings(result)
	return result
}

func (m nodeSpecMap) toGraph() *v1.NetworkGraph {
	result := &v1.NetworkGraph{
		Nodes: make([]*v1.NetworkNode, 0, len(m)),
	}
	for node, spec := range m {
		result.Nodes = append(result.Nodes, &v1.NetworkNode{
			Entity: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				Id:   node,
				Desc: &storage.NetworkEntityInfo_Deployment_{
					Deployment: &storage.NetworkEntityInfo_Deployment{
						Name: node,
					},
				},
			},
			OutEdges:  make(map[int32]*v1.NetworkEdgePropertiesBundle, len(spec.adjacencies)),
			PolicyIds: sortedIDs(spec.policies),
		})
	}
	sort.Slice(result.Nodes, func(i, j int) bool { return result.Nodes[i].Entity.Id < result.Nodes[j].Entity.Id })
	nodeIDs := make(map[string]int, len(result.Nodes))
	for idx, node := range result.Nodes {
		nodeIDs[node.Entity.Id] = idx
	}
	for node, spec := range m {
		node := result.Nodes[nodeIDs[node]]
		for _, succ := range spec.adjacencies {
			node.OutEdges[int32(nodeIDs[succ])] = &v1.NetworkEdgePropertiesBundle{}
		}
	}
	return result
}

func (m nodeSpecMap) toDiff(g *v1.NetworkGraph) *v1.NetworkGraphDiff {
	result := &v1.NetworkGraphDiff{
		NodeDiffs: make(map[string]*v1.NetworkNodeDiff, len(m)),
	}
	nodeIDs := make(map[string]int, len(g.Nodes))
	for idx, node := range g.Nodes {
		nodeIDs[node.Entity.Id] = idx
	}
	for nodeID, spec := range m {
		diff := &v1.NetworkNodeDiff{
			PolicyIds: sortedIDs(spec.policies),
			OutEdges:  make(map[string]*v1.NetworkEdgePropertiesBundle),
		}
		for _, succID := range spec.adjacencies {
			diff.OutEdges[succID] = &v1.NetworkEdgePropertiesBundle{}
		}
		result.NodeDiffs[nodeID] = diff
	}
	return result
}

func TestGraphDiffMismatchingNodes(t *testing.T) {
	t.Parallel()

	g1 := nodeSpecMap{
		"a": {},
	}.toGraph()
	g2 := nodeSpecMap{
		"a": {},
		"b": {},
	}.toGraph()

	removed, added, err := ComputeDiff(g1, g2)
	assert.NoError(t, err)
	assert.Empty(t, removed.GetNodeDiffs())
	assert.True(t, proto.Equal(nodeSpecMap{"b": {}}.toDiff(g2), added))

	g1 = nodeSpecMap{
		"a": {},
		"b": {},
	}.toGraph()
	g2 = nodeSpecMap{
		"a": {},
	}.toGraph()

	removed, added, err = ComputeDiff(g1, g2)
	assert.NoError(t, err)
	assert.Empty(t, added.GetNodeDiffs())
	assert.True(t, proto.Equal(nodeSpecMap{"b": {}}.toDiff(g1), removed))
}

func TestGraphDiffSameGraph(t *testing.T) {
	t.Parallel()

	g1 := nodeSpecMap{
		"a": {adjacencies: adjs{"b", "c"}, policies: pols{}},
		"b": {adjacencies: adjs{"c"}, policies: pols{"Pol1"}},
		"c": {adjacencies: adjs{"a"}, policies: pols{"Pol2"}},
	}.toGraph()
	g2 := g1

	removed, added, err := ComputeDiff(g1, g2)
	assert.NoError(t, err)
	assert.Empty(t, removed.GetNodeDiffs())
	assert.Empty(t, added.GetNodeDiffs())
}

func TestGraphDiffOnlyAdded(t *testing.T) {
	t.Parallel()

	g1 := nodeSpecMap{
		"a": {adjacencies: adjs{"b", "c"}, policies: pols{}},
		"b": {adjacencies: adjs{"c"}, policies: pols{"Pol1"}},
		"c": {adjacencies: adjs{"a"}, policies: pols{"Pol2"}},
	}.toGraph()
	g2 := nodeSpecMap{
		"a": {adjacencies: adjs{"b", "c"}, policies: pols{"Pol1"}},
		"b": {adjacencies: adjs{"a", "c"}, policies: pols{"Pol1"}},
		"c": {adjacencies: adjs{"a", "b"}, policies: pols{"Pol2", "Pol3"}},
	}.toGraph()
	removed, added, err := ComputeDiff(g1, g2)
	assert.NoError(t, err)
	assert.Empty(t, removed.GetNodeDiffs())

	expectedAdded := nodeSpecMap{
		"a": {policies: pols{"Pol1"}},
		"b": {adjacencies: adjs{"a"}},
		"c": {adjacencies: adjs{"b"}, policies: pols{"Pol3"}},
	}.toDiff(g1)
	assert.True(t, proto.Equal(expectedAdded, added))
}

func TestGraphDiffOnlyRemoved(t *testing.T) {
	t.Parallel()

	g1 := nodeSpecMap{
		"a": {adjacencies: adjs{"b", "c"}, policies: pols{}},
		"b": {adjacencies: adjs{"c"}, policies: pols{"Pol1"}},
		"c": {adjacencies: adjs{"a"}, policies: pols{"Pol2"}},
	}.toGraph()
	g2 := nodeSpecMap{
		"a": {adjacencies: adjs{"b", "c"}, policies: pols{"Pol1"}},
		"b": {adjacencies: adjs{"a", "c"}, policies: pols{"Pol1"}},
		"c": {adjacencies: adjs{"a", "b"}, policies: pols{"Pol2", "Pol3"}},
	}.toGraph()
	removed, added, err := ComputeDiff(g2, g1)
	assert.NoError(t, err)
	assert.Empty(t, added.GetNodeDiffs())

	expectedRemoved := nodeSpecMap{
		"a": {policies: pols{"Pol1"}},
		"b": {adjacencies: adjs{"a"}},
		"c": {adjacencies: adjs{"b"}, policies: pols{"Pol3"}},
	}.toDiff(g1)
	assert.True(t, proto.Equal(expectedRemoved, removed))
}

func TestGraphDiffAddedAndRemoved(t *testing.T) {
	t.Parallel()

	g1 := nodeSpecMap{
		"a": {adjacencies: adjs{"b", "c"}, policies: pols{}},
		"b": {adjacencies: adjs{"c"}, policies: pols{"Pol1"}},
		"c": {adjacencies: adjs{"a"}, policies: pols{"Pol2"}},
		"d": {adjacencies: adjs{"a", "f"}, policies: pols{"Pol3"}},
		"e": {adjacencies: adjs{"b", "c", "d"}, policies: pols{"Pol3", "Pol4"}},
		"f": {adjacencies: adjs{"a", "b", "c", "d", "e"}, policies: pols{"Pol1", "Pol2"}},
		"g": {adjacencies: adjs{"f"}, policies: pols{"Pol1"}},
		"h": {adjacencies: adjs{"b", "d", "f"}, policies: pols{"Pol2"}},
		"i": {adjacencies: adjs{"a", "c", "g"}, policies: pols{"Pol1", "Pol4"}},
	}.toGraph()
	g2 := nodeSpecMap{
		"a": {adjacencies: adjs{"b", "c"}, policies: pols{"Pol3"}},                 // only policy added
		"b": {adjacencies: adjs{"c"}, policies: pols{}},                            // only policy removed
		"c": {adjacencies: adjs{"a", "e", "f"}, policies: pols{"Pol2"}},            // only adjacencies added
		"d": {adjacencies: adjs{"a"}, policies: pols{"Pol3"}},                      // only adjacencies removed
		"e": {adjacencies: adjs{"b", "c", "d", "g"}, policies: pols{"Pol4"}},       // adjacency added, policy removed
		"f": {adjacencies: adjs{"a", "e"}, policies: pols{"Pol1", "Pol2", "Pol4"}}, // adjacencies removed, policy added
		"g": {adjacencies: adjs{"f", "h"}, policies: pols{"Pol1", "Pol3"}},         // adjacency and policy added
		"h": {adjacencies: adjs{}, policies: pols{}},                               // adjacencies and policies removed
		"i": {adjacencies: adjs{"a", "c", "g"}, policies: pols{"Pol1", "Pol4"}},    // unchanged
	}.toGraph()
	removed, added, err := ComputeDiff(g1, g2)
	assert.NoError(t, err)

	expectedRemoved := nodeSpecMap{
		"b": {policies: pols{"Pol1"}},
		"d": {adjacencies: adjs{"f"}},
		"e": {policies: pols{"Pol3"}},
		"f": {adjacencies: adjs{"b", "c", "d"}},
		"h": {adjacencies: adjs{"b", "d", "f"}, policies: pols{"Pol2"}},
	}.toDiff(g1)
	expectedAdded := nodeSpecMap{
		"a": {policies: pols{"Pol3"}},
		"c": {adjacencies: adjs{"e", "f"}},
		"e": {adjacencies: adjs{"g"}},
		"f": {policies: pols{"Pol4"}},
		"g": {adjacencies: adjs{"h"}, policies: pols{"Pol3"}},
	}.toDiff(g1)
	assert.True(t, proto.Equal(expectedRemoved, removed))
	assert.True(t, proto.Equal(expectedAdded, added))
}
