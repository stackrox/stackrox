package dependency

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/common/ingestion"
)

type graphNode struct {
	kind string
	// canonicalId for children
	dependencies set.StringSet
	// canonicalId for parents
	dependants set.StringSet
}

type Graph struct {
	stores *ingestion.ResourceStore
	nodeIndex map[string]*graphNode
}

func NewGraph(stores *ingestion.ResourceStore) *Graph {
	return &Graph{
		stores: stores,
		nodeIndex: map[string]*graphNode{},
	}
}

type SnapshotNode struct {
	Kind string
	Object interface{}
	Children []*SnapshotNode
}

type ClusterSnapshot struct {
	TopLevelNodes []*SnapshotNode
}

// GenerateSnapshotFromUpsert processes a create or update to the graph and returns a snapshot of
// every cluster segment affected by this update.
// Empty namespace means that this is a global level resource.
// It returns a tree with a top level resource that needs to be updated and all its dependencies.
// For example, if a snapshot for a Role is being created, the dependency graph will look for
// the top-most parent of this role and return all the dependencies under it. If the top-level
// parent is a deployment, it might return a snapshot looking like:
//   [Deployment] --> [Binding] -> [Role*]
//               \
//                `--> [Network Policy]
//  *Role is the deployment that triggered the snapshot
//
// If the edges changed on an update, this snapshot will contain both the old and the new
// top-level linked to the updated resource.
func (g *Graph) GenerateSnapshotFromUpsert(kind, namespace, id string) *ClusterSnapshot {
	identifier := Identifier{
		Kind:      kind,
		Namespace: namespace,
		Id:        id,
	}

	canonicalId := makeCanonicalId(identifier)
	if _, exists := g.nodeIndex[canonicalId]; exists {
		// TODO: check if edges were updated and then re-compute the graph for that node
	} else {
		g.addNodeAndAdjacentIfMissing(identifier)
	}

	// On a regular update or create, we want to find the top-level node, and genreate a snapshot from there.
	var nodes []string
	g.forEachTopLevelNode(canonicalId, func(s string) {
		nodes = append(nodes, s)
	})
	return g.makeSnapshotFrom(nodes)
}

func (g *Graph) addNodeAndAdjacentIfMissing(identifier Identifier) {
	canonicalId := makeCanonicalId(identifier)
	resourceObject := g.getResourceById(identifier.Kind, identifier.Id)
	if _, exists := g.nodeIndex[canonicalId]; exists {
		// node already exists, and it's being updated, we can skip processing
		return
	} else {
		// node doesn't exist, we need to add it to the graph
		g.nodeIndex[canonicalId] = &graphNode{
			kind: identifier.Kind,
		}

		// since this is being created now, we need to check if there are already resources
		// in the cluster that relate to this. If there are, these resources might also
		// need to be added to the graph if they're missing.

		finder, ok := FinderForKind[identifier.Kind]
		if !ok {
			panic(fmt.Sprintf("finder for kind %s not implemented", identifier.Kind))
		}

		for _, dependency := range finder.FindDependencies(resourceObject, g.stores) {
			g.addNodeAndAdjacentIfMissing(dependency)
			childId := makeCanonicalId(dependency)

			// Add edges on both nodes
			g.nodeIndex[canonicalId].dependencies.Add(childId)
			g.nodeIndex[childId].dependants.Add(canonicalId)
		}

		for _, dependant := range finder.FindDependants(resourceObject, g.stores) {
			g.addNodeAndAdjacentIfMissing(dependant)
			parentId := makeCanonicalId(dependant)

			// Add edges on both nodes
			g.nodeIndex[canonicalId].dependants.Add(parentId)
			g.nodeIndex[parentId].dependencies.Add(canonicalId)
		}
	}
}

func (g *Graph) forEachTopLevelNode(canonicalId string, fn func(string)) {
	if len(g.nodeIndex[canonicalId].dependants) == 0 {
		fn(canonicalId)
	} else {
		for _, dep := range g.nodeIndex[canonicalId].dependants.AsSlice() {
			g.forEachTopLevelNode(dep, fn)
		}
	}
}

func (g *Graph) makeSnapshotFrom(ids []string) *ClusterSnapshot {
	snapshot := new(ClusterSnapshot)
	for _, id := range ids {
		snapshot.TopLevelNodes = append(snapshot.TopLevelNodes, g.makeSnapshot(id))
	}
	return snapshot
}

func (g *Graph) makeSnapshot(id string) *SnapshotNode {
	parts := strings.Split(id, "#")
	rawObject := g.getResourceById(parts[0], parts[2])
	node := g.nodeIndex[id]
	var children []*SnapshotNode
	for _, dependency := range node.dependencies.AsSlice() {
		children = append(children, g.makeSnapshot(dependency))
	}

	return &SnapshotNode{
		Kind:     parts[0],
		Object:   rawObject,
		Children: children,
	}
}

// GenerateSnapshotFromDelete processes a deletion event to the graph and returns a snapshot of
// every cluster segment affected by this update.
// This work similarly to GenerateSnapshotFromUpsert. But it removes the edges to and from the
// node after the snapshot is processed.
func (g *Graph) GenerateSnapshotFromDelete(kind, namespace, id string) *ClusterSnapshot {
	return &ClusterSnapshot{}
}

func (g *Graph) getResourceById(kind string, id string) interface{} {
	switch kind {
	case "Deployment":
		return g.stores.Deployments.Get(id)
	case "Pod":
		// TODO: name? I think this should also be indexed by its id?
		return g.stores.PodStore.GetByName(kind, id)
	case "NetworkPolicy":
		return g.stores.NetworkPolicy.Get(id)
	default:
		// TODO: return error
		return nil
	}
}


func makeCanonicalId(i Identifier) string {
	return fmt.Sprintf("%s#%s#%s", i.Kind, i.Namespace, i.Id)
}
