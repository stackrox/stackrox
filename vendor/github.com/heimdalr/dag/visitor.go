package dag

import (
	"sort"

	llq "github.com/emirpasic/gods/queues/linkedlistqueue"
	lls "github.com/emirpasic/gods/stacks/linkedliststack"
)

// Visitor is the interface that wraps the basic Visit method.
// It can use the Visitor and XXXWalk functions together to traverse the entire DAG.
// And access per-vertex information when traversing.
type Visitor interface {
	Visit(Vertexer)
}

// DFSWalk implements the Depth-First-Search algorithm to traverse the entire DAG.
// The algorithm starts at the root node and explores as far as possible
// along each branch before backtracking.
func (d *DAG) DFSWalk(visitor Visitor) {
	d.muDAG.RLock()
	defer d.muDAG.RUnlock()

	stack := lls.New()

	vertices := d.getRoots()
	for _, id := range reversedVertexIDs(vertices) {
		v := vertices[id]
		sv := storableVertex{WrappedID: id, Value: v}
		stack.Push(sv)
	}

	visited := make(map[string]bool, d.getSize())

	for !stack.Empty() {
		v, _ := stack.Pop()
		sv := v.(storableVertex)

		if !visited[sv.WrappedID] {
			visited[sv.WrappedID] = true
			visitor.Visit(sv)
		}

		vertices, _ := d.getChildren(sv.WrappedID)
		for _, id := range reversedVertexIDs(vertices) {
			v := vertices[id]
			sv := storableVertex{WrappedID: id, Value: v}
			stack.Push(sv)
		}
	}
}

// BFSWalk implements the Breadth-First-Search algorithm to traverse the entire DAG.
// It starts at the tree root and explores all nodes at the present depth prior
// to moving on to the nodes at the next depth level.
func (d *DAG) BFSWalk(visitor Visitor) {
	d.muDAG.RLock()
	defer d.muDAG.RUnlock()

	queue := llq.New()

	vertices := d.getRoots()
	for _, id := range vertexIDs(vertices) {
		v := vertices[id]
		sv := storableVertex{WrappedID: id, Value: v}
		queue.Enqueue(sv)
	}

	visited := make(map[string]bool, d.getOrder())

	for !queue.Empty() {
		v, _ := queue.Dequeue()
		sv := v.(storableVertex)

		if !visited[sv.WrappedID] {
			visited[sv.WrappedID] = true
			visitor.Visit(sv)
		}

		vertices, _ := d.getChildren(sv.WrappedID)
		for _, id := range vertexIDs(vertices) {
			v := vertices[id]
			sv := storableVertex{WrappedID: id, Value: v}
			queue.Enqueue(sv)
		}
	}
}

func vertexIDs(vertices map[string]interface{}) []string {
	ids := make([]string, 0, len(vertices))
	for id := range vertices {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func reversedVertexIDs(vertices map[string]interface{}) []string {
	ids := vertexIDs(vertices)
	i, j := 0, len(ids)-1
	for i < j {
		ids[i], ids[j] = ids[j], ids[i]
		i++
		j--
	}
	return ids
}
