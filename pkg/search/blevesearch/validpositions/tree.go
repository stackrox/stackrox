package validpositions

// A Tree represents a tree of valid array positions. It's a data structure very specifically designed to store
// valid Bleve array positions, and computing their intersections.
// An array position is a []uint64 that denotes the array positions of a certain match
// Example: if we have a field deployment.containers.volumes.name
// and this is matched a volume name "vol1",
// then an array position of []uint64{1, 2} denotes that the match was on the object corresponding to
// deployment.GetContainers()[1].GetVolumes()[2].GetName()
// This tree data structure helps us to match only fields that have the same array positions.
type Tree struct {
	root      *node
	maxLength int
	nonEmpty  bool
}

type node struct {
	children map[uint64]*node
}

// NewTreeFromValues returns a new tree from the given values.
func NewTreeFromValues(valueSlices ...[]uint64) *Tree {
	tree := NewTree()
	for _, valueSlice := range valueSlices {
		tree.Add(valueSlice)
	}
	return tree
}

// NewTree returns a ready-to-use tree.
func NewTree() *Tree {
	return &Tree{root: newNode()}
}

func newNode() *node {
	return &node{children: make(map[uint64]*node)}
}

// Empty returns whether the tree is empty.
func (t *Tree) Empty() bool {
	return t == nil || !t.nonEmpty
}

// Add adds the given values to the tree.
func (t *Tree) Add(values []uint64) {
	t.nonEmpty = true
	if len(values) > t.maxLength {
		t.maxLength = len(values)
	}
	t.root.add(values)
}

func (n *node) add(values []uint64) {
	if len(values) == 0 {
		return
	}
	node, ok := n.children[values[0]]
	if !ok {
		node = newNode()
		n.children[values[0]] = node
	}
	node.add(values[1:])
}

// Merge merges the two trees. It essentially computes their intersection, only leaving behind
// paths that exist in both the trees.
// The tree is modified in-place. There are no guarantees that the "other" will not be touched.
func (t *Tree) Merge(other *Tree) {
	if t.Empty() {
		return
	}
	if other.Empty() {
		t.root = newNode()
		t.nonEmpty = false
		return
	}
	t.root.merge(other.root)
}

func (n *node) merge(other *node) {
	// If the other doesn't have any children, then its length is simply shorter than this one.
	// This is fine, and doesn't affect our intersection.
	// For example, if one element has array positions [1, 2] and the other has array positions [1, 2, 3]
	// then they _are_ matching -- it's just that one element is more nested than the other.
	if len(other.children) == 0 {
		return
	}
	// In this case, this tree is shorter than the other tree, so we copy the other's children over.
	if len(n.children) == 0 {
		n.children = other.children
		return
	}
	for val, child := range n.children {
		otherChild, exists := other.children[val]
		if !exists {
			delete(n.children, val)
			continue
		}
		child.merge(otherChild)
	}
}

// Contains returns whether the tree contains the given set of values.
func (t *Tree) Contains(values []uint64) bool {
	if t == nil {
		return false
	}
	return t.root.contains(values)
}

func (n *node) contains(values []uint64) bool {
	if len(values) == 0 {
		return true
	}
	node, ok := n.children[values[0]]
	if !ok {
		return false
	}
	return node.contains(values[1:])
}
