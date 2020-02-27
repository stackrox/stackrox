package trie

import (
	"sort"
)

// Tri is a structure for storing key, value pairs in a tree for lookup.
type Tri interface {
	Insert(key []byte)
	Contains(key []byte) bool
}

// New returns a new instance of a Tri.
func New() Tri {
	return &rootTrie{}
}

type rootTrie struct {
	node trieNode
}

// Insert an index and key into the tri.
func (root *rootTrie) Insert(key []byte) {
	root.node = root.node.insertRec(key)
}

func (root *rootTrie) Contains(key []byte) bool {
	return root.node.contains(key)
}

type trieNode struct {
	key   []byte
	isKey bool

	branch []trieNode
}

func (node trieNode) insertRec(key []byte) trieNode {
	if len(key) == 0 {
		return node
	}

	dist := getPrefixDist(node.key, key)
	if dist < len(key) && dist < len(node.key) {
		return trieNode{
			key: key[:dist],
			branch: order(
				trieNode{
					key:   key[dist:],
					isKey: true,
				},
				trieNode{
					key:    node.key[dist:],
					isKey:  node.isKey,
					branch: node.branch,
				},
			),
		}
	} else if dist < len(node.key) {
		return trieNode{
			key:   key,
			isKey: true,
			branch: []trieNode{
				{
					key:    node.key[dist:],
					isKey:  node.isKey,
					branch: node.branch,
				},
			},
		}
	} else if dist < len(key) {
		pos, matches := findPosition(node.branch, key[dist:])
		if matches {
			node.branch[pos] = node.branch[pos].insertRec(key[dist:])
		} else {
			node.branch = insertAt(node.branch, pos, key[dist:])
		}
	}
	return node
}

// Contains returns if the trie contains the key.
func (node trieNode) contains(key []byte) bool {
	dist := getPrefixDist(node.key, key)
	if dist < len(node.key) {
		return false
	} else if dist == len(node.key) && dist == len(key) {
		return node.isKey
	}

	pos, matches := findPosition(node.branch, key[dist:])
	if !matches {
		return false
	}
	return node.branch[pos].contains(key[dist:])
}

// Returns the position of the node that has a shared prefix with the given key if one exists. If not, the returned
// value of dist is 0.
func findPosition(keys []trieNode, key []byte) (pos int, matches bool) {
	pos = getPos(keys, key)
	if len(keys) > 0 && pos < len(keys) {
		matches = keys[pos].key[0] == key[0]
	}
	return pos, matches
}

// Returns the position in the given array of nodes where the given key would fit based on first byte.
func getPos(keys []trieNode, key []byte) int {
	if len(keys) == 0 {
		return 0
	}
	return sort.Search(len(keys), func(i int) bool {
		return key[0] <= keys[i].key[0]
	})
}

// Inserts a node in the given position with the given key and value.
func insertAt(keys []trieNode, pos int, key []byte) []trieNode {
	newNode := trieNode{
		key:   key,
		isKey: true,
	}
	if len(keys) == 0 {
		return []trieNode{newNode}
	}
	return append(keys[:pos], append([]trieNode{newNode}, keys[pos:]...)...)
}

// Return the two  input nodes in the order that they should be set in the branch list.
func order(n1, n2 trieNode) []trieNode {
	if len(n1.key) == 0 {
		return []trieNode{n2}
	} else if len(n2.key) == 0 {
		return []trieNode{n1}
	}
	if n1.key[0] < n2.key[0] {
		return []trieNode{n1, n2}
	}
	return []trieNode{n2, n1}
}

// Returns the number of bytes that represent a common prefix.
// For instance: []byte{0, 1, 2, 3}, and []byte{0, 1, 4, 5}, share the first 2 bytes, so this would return 2.
func getPrefixDist(k1, k2 []byte) int {
	dist := 0
	for dist < len(k1) && dist < len(k2) && k1[dist] == k2[dist] {
		dist++
	}
	return dist
}
