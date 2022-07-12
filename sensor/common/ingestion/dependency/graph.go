package dependency

import "github.com/stackrox/rox/sensor/common/ingestion"

type Graph struct {
	stores *ingestion.ResourceStore
}

func NewGraph(stores *ingestion.ResourceStore) *Graph {
	return &Graph{
		stores: stores,
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
	return &ClusterSnapshot{}
}

// GenerateSnapshotFromDelete processes a deletion event to the graph and returns a snapshot of
// every cluster segment affected by this update.
// This work similarly to GenerateSnapshotFromUpsert. But it removes the edges to and from the
// node after the snapshot is processed.
func (g *Graph) GenerateSnapshotFromDelete(kind, namespace, id string) *ClusterSnapshot {
	return &ClusterSnapshot{}
}
